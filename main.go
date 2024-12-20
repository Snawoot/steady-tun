package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	conn "github.com/Snawoot/steady-tun/conn"
	"github.com/Snawoot/steady-tun/dnscache"
	clog "github.com/Snawoot/steady-tun/log"
	"github.com/Snawoot/steady-tun/pool"
	"github.com/Snawoot/steady-tun/server"
)

var (
	version = "undefined"
)

func perror(msg string) {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, msg)
}

func arg_fail(msg string) {
	perror(msg)
	perror("Usage:")
	flag.PrintDefaults()
	os.Exit(2)
}

type CLIArgs struct {
	host                  string
	port                  uint
	verbosity             int
	bind_address          string
	bind_port             uint
	pool_size             uint
	dialers               uint
	backoff, ttl, timeout time.Duration
	cert, key, cafile     string
	hostname_check        bool
	tls_servername        string
	tlsSessionCache       bool
	tlsEnabled            bool
	dnsCacheTTL           time.Duration
	dnsNegCacheTTL        time.Duration
	showVersion           bool
}

func parse_args() CLIArgs {
	args := CLIArgs{}
	flag.StringVar(&args.host, "dsthost", "", "destination server hostname")
	flag.UintVar(&args.port, "dstport", 0, "destination server port")
	flag.IntVar(&args.verbosity, "verbosity", 20, "logging verbosity "+
		"(10 - debug, 20 - info, 30 - warning, 40 - error, 50 - critical)")
	flag.StringVar(&args.bind_address, "bind-address", "127.0.0.1", "bind address")
	flag.UintVar(&args.bind_port, "bind-port", 57800, "bind port")
	flag.UintVar(&args.pool_size, "pool-size", 50, "connection pool size")
	flag.UintVar(&args.dialers, "dialers", uint(4*runtime.GOMAXPROCS(0)), "concurrency limit for TLS connection attempts")
	flag.DurationVar(&args.backoff, "backoff", 5*time.Second, "delay between connection attempts")
	flag.DurationVar(&args.ttl, "ttl", 30*time.Second, "lifetime of idle pool connection in seconds")
	flag.DurationVar(&args.timeout, "timeout", 4*time.Second, "server connect timeout")
	flag.StringVar(&args.cert, "cert", "", "use certificate for client TLS auth")
	flag.StringVar(&args.key, "key", "", "key for TLS certificate")
	flag.StringVar(&args.cafile, "cafile", "", "override default CA certs by specified in file")
	flag.BoolVar(&args.hostname_check, "hostname-check", true, "check hostname in server cert subject")
	flag.StringVar(&args.tls_servername, "tls-servername", "", "specifies hostname to expect in server cert")
	flag.BoolVar(&args.tlsSessionCache, "tls-session-cache", true, "enable TLS session cache")
	flag.BoolVar(&args.showVersion, "version", false, "show program version and exit")
	flag.BoolVar(&args.tlsEnabled, "tls-enabled", true, "enable TLS client for pool connections")
	flag.DurationVar(&args.dnsCacheTTL, "dns-cache-ttl", 30*time.Second, "DNS cache TTL")
	flag.DurationVar(&args.dnsNegCacheTTL, "dns-neg-cache-ttl", 1*time.Second, "negative DNS cache TTL")
	flag.Parse()
	if args.showVersion {
		return args
	}
	if args.host == "" {
		arg_fail("Destination host argument is required!")
	}
	if args.port == 0 {
		arg_fail("Destination host argument is required!")
	}
	if args.port >= 65536 {
		arg_fail("Bad destination port!")
	}
	if args.bind_port >= 65536 {
		arg_fail("Bad bind port!")
	}
	if args.dialers < 1 {
		arg_fail("dialers parameter should be not less than 1")
	}
	return args
}

func main() {
	args := parse_args()
	if args.showVersion {
		fmt.Println(version)
		return
	}

	logWriter := clog.NewLogWriter(os.Stderr)
	defer logWriter.Close()

	mainLogger := clog.NewCondLogger(log.New(logWriter, "MAIN    : ", log.LstdFlags|log.Lshortfile),
		args.verbosity)
	listenerLogger := clog.NewCondLogger(log.New(logWriter, "LISTENER: ", log.LstdFlags|log.Lshortfile),
		args.verbosity)
	handlerLogger := clog.NewCondLogger(log.New(logWriter, "HANDLER : ", log.LstdFlags|log.Lshortfile),
		args.verbosity)
	connLogger := clog.NewCondLogger(log.New(logWriter, "CONN    : ", log.LstdFlags|log.Lshortfile),
		args.verbosity)
	poolLogger := clog.NewCondLogger(log.New(logWriter, "POOL    : ", log.LstdFlags|log.Lshortfile),
		args.verbosity)

	var (
		dialer      conn.ContextDialer
		connfactory conn.Factory
		err         error
	)
	dialer = (&net.Dialer{
		Timeout: args.timeout,
	}).DialContext

	if args.dnsCacheTTL > 0 {
		dialer = dnscache.WrapDialer(dialer, net.DefaultResolver, 128, args.dnsCacheTTL, args.dnsNegCacheTTL, args.timeout)
	}

	if args.tlsEnabled {
		var sessionCache tls.ClientSessionCache
		if args.tlsSessionCache {
			sessionCache = tls.NewLRUClientSessionCache(2 * int(args.pool_size))
		}
		connfactory, err = conn.NewTLSConnFactory(args.host,
			uint16(args.port),
			dialer,
			args.cert,
			args.key,
			args.cafile,
			args.hostname_check,
			args.tls_servername,
			args.dialers,
			sessionCache,
			connLogger)
		if err != nil {
			panic(err)
		}
	} else {
		connfactory = conn.NewPlainConnFactory(args.host, uint16(args.port), dialer)
	}
	connPool := pool.NewConnPool(args.pool_size, args.ttl, args.backoff, connfactory.DialContext, poolLogger)
	connPool.Start()
	defer connPool.Stop()

	listener := server.NewTCPListener(args.bind_address,
		uint16(args.bind_port),
		server.NewConnHandler(connPool, handlerLogger).Handle,
		listenerLogger)
	if err := listener.Start(); err != nil {
		panic(err)
	}
	defer listener.Stop()

	mainLogger.Info("Listener started.")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	mainLogger.Info("Shutting down...")
}
