package conn

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"

	"golang.org/x/sync/semaphore"

	clog "github.com/Snawoot/steady-tun/log"
)

type TLSConnFactory struct {
	addr      string
	tlsConfig *tls.Config
	dialer    ContextDialer
	sem       *semaphore.Weighted
}

var _ Factory = &TLSConnFactory{}

func NewTLSConnFactory(host string, port uint16, dialer ContextDialer,
	certfile, keyfile string, cafile string, hostname_check bool,
	tls_servername string, dialers uint, sessionCache tls.ClientSessionCache, logger *clog.CondLogger) (*TLSConnFactory, error) {
	if !hostname_check && cafile == "" {
		return nil, errors.New("Hostname check should not be disabled in absence of custom CA file")
	}
	if certfile != "" && keyfile == "" || certfile == "" && keyfile != "" {
		return nil, errors.New("Certificate file and key file must be specified only together")
	}
	var certs []tls.Certificate
	if certfile != "" && keyfile != "" {
		cert, err := tls.LoadX509KeyPair(certfile, keyfile)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
	var roots *x509.CertPool
	if cafile != "" {
		roots = x509.NewCertPool()
		certs, err := ioutil.ReadFile(cafile)
		if err != nil {
			return nil, err
		}
		if ok := roots.AppendCertsFromPEM(certs); !ok {
			return nil, errors.New("Failed to load CA certificates")
		}
	}
	servername := host
	if tls_servername != "" {
		servername = tls_servername
	}
	tlsConfig := tls.Config{
		RootCAs:            roots,
		ServerName:         servername,
		Certificates:       certs,
		ClientSessionCache: sessionCache,
	}
	if !hostname_check {
		tlsConfig.InsecureSkipVerify = true
		tlsConfig.VerifyPeerCertificate = func(certificates [][]byte, _ [][]*x509.Certificate) error {
			certs := make([]*x509.Certificate, len(certificates))
			for i, asn1Data := range certificates {
				cert, err := x509.ParseCertificate(asn1Data)
				if err != nil {
					return errors.New("tls: failed to parse certificate from server: " + err.Error())
				}
				certs[i] = cert
			}

			opts := x509.VerifyOptions{
				Roots:         roots, // On the server side, use config.ClientCAs.
				DNSName:       "",    // No hostname check
				Intermediates: x509.NewCertPool(),
			}
			for _, cert := range certs[1:] {
				opts.Intermediates.AddCert(cert)
			}
			_, err := certs[0].Verify(opts)
			return err
		}
	}
	return &TLSConnFactory{
		addr:      net.JoinHostPort(host, strconv.Itoa(int(port))),
		tlsConfig: &tlsConfig,
		dialer:    dialer,
		sem:       semaphore.NewWeighted(int64(dialers)),
	}, nil
}

func (cf *TLSConnFactory) DialContext(ctx context.Context) (net.Conn, error) {
	if cf.sem.Acquire(ctx, 1) != nil {
		return nil, errors.New("Context was cancelled")
	}
	defer cf.sem.Release(1)
	netConn, err := cf.dialer(ctx, "tcp", cf.addr)
	if err != nil {
		return nil, fmt.Errorf("cf.dialer.DialContext(ctx, \"tcp\", %q) failed: %v", cf.addr, err)
	}
	tlsConn := tls.Client(netConn, cf.tlsConfig)
	err = tlsConn.HandshakeContext(ctx)
	if err != nil {
		netConn.Close()
		return nil, fmt.Errorf("tlsConn.HandshakeContext(ctx) failed: %v", err)
	}
	return tlsConn, nil
}
