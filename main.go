package main

import (
    "log"
    "os"
    "fmt"
    "flag"
    "os/signal"
    "syscall"
    "time"
    "runtime"
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
    host string
    port uint
    verbosity int
    bind_address string
    bind_port uint
    pool_size uint
    dialers uint
    backoff, ttl, timeout, pool_wait time.Duration
    cert, key, cafile string
    hostname_check bool
    tls_servername string
    flagset *flag.FlagSet
}


func NewCLIArgs() *CLIArgs {
    args := CLIArgs{flagset: flag.NewFlagSet(os.Args[0], flag.ContinueOnError)}
    args.flagset.StringVar(&args.host, "dsthost", "", "destination server hostname")
    args.flagset.UintVar(&args.port, "dstport", 0, "destination server port")
    args.flagset.IntVar(&args.verbosity, "verbosity", 20, "logging verbosity " +
                        "(10 - debug, 20 - info, 30 - warning, 40 - error, 50 - critical)")
    args.flagset.StringVar(&args.bind_address, "bind-address", "127.0.0.1", "bind address")
    args.flagset.UintVar(&args.bind_port, "bind-port", 57800, "bind port")
    args.flagset.UintVar(&args.pool_size, "pool-size", 50, "connection pool size")
    args.flagset.UintVar(&args.dialers, "dialers", uint(runtime.GOMAXPROCS(0)), "concurrency limit for TLS connection attempts")
    args.flagset.DurationVar(&args.backoff, "backoff", 5 * time.Second, "delay between connection attempts")
    args.flagset.DurationVar(&args.ttl, "ttl", 30 * time.Second, "lifetime of idle pool connection in seconds")
    args.flagset.DurationVar(&args.timeout, "timeout", 4 * time.Second, "server connect timeout")
    args.flagset.DurationVar(&args.pool_wait, "pool-wait", 15 * time.Second, "timeout for acquiring connection from pool")
    args.flagset.StringVar(&args.cert, "cert", "", "use certificate for client TLS auth")
    args.flagset.StringVar(&args.key, "key", "", "key for TLS certificate")
    args.flagset.StringVar(&args.cafile, "cafile", "", "override default CA certs by specified in file")
    args.flagset.BoolVar(&args.hostname_check, "hostname-check", true, "check hostname in server cert subject")
    args.flagset.StringVar(&args.tls_servername, "tls-servername", "", "specifies hostname to expect in server cert")
    return &args
}

func parse_args() *CLIArgs {
    args := NewCLIArgs()
    pluginArgs, err := NewPluginArgs()
    if err == nil {
        opts := pluginArgs.ExportOptions()

        if err := args.Update(opts); err != nil {
            log.Printf("main: WARNING: CLIArgs.Update: %v", err)
        }

        if args.Server {
            // args.Dst = pluginArgs.GetLocalAddr()
            // args.Bind = pluginArgs.GetRemoteAddr()
        } else {
            // args.Bind = pluginArgs.GetLocalAddr()
            // args.Dst = pluginArgs.GetRemoteAddr()
        }
    }
    err = args.Update(os.Args[1:])
    if err != nil {
        log.Fatalf("main: args.Update: %v", err)
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

    logWriter := NewLogWriter(os.Stderr)
    defer logWriter.Close()

    mainLogger := NewCondLogger(log.New(logWriter, "MAIN    : ", log.LstdFlags | log.Lshortfile),
                                args.verbosity)
    listenerLogger := NewCondLogger(log.New(logWriter, "LISTENER: ", log.LstdFlags | log.Lshortfile),
                                    args.verbosity)
    handlerLogger := NewCondLogger(log.New(logWriter, "HANDLER : ", log.LstdFlags | log.Lshortfile),
                                   args.verbosity)
    connLogger := NewCondLogger(log.New(logWriter, "CONN    : ", log.LstdFlags | log.Lshortfile),
                                args.verbosity)
    poolLogger := NewCondLogger(log.New(logWriter, "POOL    : ", log.LstdFlags | log.Lshortfile),
                                args.verbosity)

    connfactory, err := NewConnFactory(args.host,
                                       uint16(args.port),
                                       args.timeout,
                                       args.cert,
                                       args.key,
                                       args.cafile,
                                       args.hostname_check,
                                       args.tls_servername,
                                       args.dialers,
                                       connLogger)
    if err != nil {
        panic(err)
    }
    pool := NewConnPool(args.pool_size, args.ttl, args.backoff, connfactory, poolLogger)
    pool.Start()
    defer pool.Stop()

    listener := NewTCPListener(args.bind_address,
                               uint16(args.bind_port),
                               NewConnHandler(pool, args.pool_wait, handlerLogger).handle,
                               listenerLogger)
    if err:= listener.Start(); err != nil {
        panic(err)
    }
    defer listener.Stop()

    mainLogger.Info("Listener started.")
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
    <-sigs
    mainLogger.Info("Shutting down...")
}
