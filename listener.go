package main

import (
	"net"
)

type TCPListener struct {
    address string
    port uint16
    handler func(*net.TCPConn)
    quit chan bool
    listener *net.TCPListener
    logger *CondLogger
}

func NewTCPListener(address string, port uint16, handler func(*net.TCPConn),
                    logger *CondLogger) *TCPListener {
    return &TCPListener{
        address: address,
        port: port,
        handler: handler,
        logger: logger,
        quit: make(chan bool, 1),
    }
}

func (l *TCPListener) Start() error {
    ips, err := net.LookupIP(l.address)
    if err != nil {
        return err
    }
    listener, err := net.ListenTCP("tcp", &net.TCPAddr{
        IP: ips[0],
        Port: int(l.port),
    })
    if err != nil {
        return err
    }
    l.listener = listener
    go l.serve()
    return nil
}

func (l *TCPListener) serve() {
    for {
        conn, err := l.listener.AcceptTCP()
        if err != nil {
            select {
            case <-l.quit:
                l.logger.Info("Leaving accept loop.")
                l.quit <- true
                return
            default:
                l.logger.Error("Accept error: %s", err)
                continue
            }
        }
        go func(c *net.TCPConn) {
            defer c.Close()
            l.handler(c)
        }(conn)
    }
}

func (l *TCPListener) Stop() {
    l.quit <- true
    l.listener.Close()
    <-l.quit
}
