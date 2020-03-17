package main

import (
    "time"
    "net"
    "crypto/tls"
    "crypto/x509"
    "io/ioutil"
    "errors"
    "context"
    "strconv"
)

type ConnFactory struct {
    addr string
    tlsConfig *tls.Config
    dialer *net.Dialer
    logger *CondLogger
}

func NewConnFactory(host string, port uint16, timeout time.Duration,
                    certfile, keyfile string, cafile string, hostname_check bool,
                    tls_servername string, logger *CondLogger) (*ConnFactory, error) {
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
    if cafile == "" {
        sysroots, err := x509.SystemCertPool()
        if err != nil {
            return nil, err
        }
        roots = sysroots
    } else {
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
        RootCAs: roots,
        ServerName: servername,
        Certificates: certs,
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
                DNSName:       "", // No hostname check
                Intermediates: x509.NewCertPool(),
            }
            for _, cert := range certs[1:] {
                opts.Intermediates.AddCert(cert)
            }
            _, err := certs[0].Verify(opts)
            return err
        }
    }
    return &ConnFactory{
        addr: net.JoinHostPort(host, strconv.Itoa(int(port))),
        tlsConfig: &tlsConfig,
        dialer: &net.Dialer{Timeout: timeout},
        logger: logger,
    }, nil
}

type WrappedConn struct {
    conn *tls.Conn
    readch chan []byte
    logger *CondLogger
}

func (cf *ConnFactory) DialContext(ctx context.Context) (*WrappedConn, error) {
    var newconn *tls.Conn
    var err error
    ch := make(chan bool, 1)
    go func () {
        newconn, err = tls.DialWithDialer(cf.dialer, "tcp", cf.addr, cf.tlsConfig)
        ch <- true
    }()
    select {
    case <- ch:
        if err != nil {
            cf.logger.Error("Got error during connection: %s", err)
            return nil, err
        }
        return &WrappedConn{conn: newconn, logger: cf.logger}, nil
    case <- ctx.Done():
        return nil, errors.New("Context was cancelled")
    }
}
