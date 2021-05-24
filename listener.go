package main

import (
	"context"
	"net"
	"sync"
)

type HandlerFunc func(context.Context, net.Conn)

type TCPListener struct {
	address    string
	port       uint16
	handler    HandlerFunc
	quitaccept chan struct{}
	listener   *net.TCPListener
	logger     *CondLogger
	ctx        context.Context
	cancel     context.CancelFunc
	shutdown   sync.WaitGroup
}

func NewTCPListener(address string, port uint16, handler HandlerFunc,
	logger *CondLogger) *TCPListener {
	ctx, cancel := context.WithCancel(context.Background())
	return &TCPListener{
		address:    address,
		port:       port,
		handler:    handler,
		logger:     logger,
		quitaccept: make(chan struct{}, 1),
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (l *TCPListener) Start() error {
	ips, err := net.LookupIP(l.address)
	if err != nil {
		return err
	}
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   ips[0],
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
		conn, err := l.listener.Accept()
		if err != nil {
			select {
			case <-l.quitaccept:
				l.logger.Info("Leaving accept loop.")
				l.quitaccept <- struct{}{}
				return
			default:
				l.logger.Error("Accept error: %s", err)
				continue
			}
		}
		l.shutdown.Add(1)
		go func(c net.Conn) {
			defer l.shutdown.Done()
			l.handler(l.ctx, c)
		}(conn)
	}
}

func (l *TCPListener) Stop() {
	l.quitaccept <- struct{}{}
	l.listener.Close()
	<-l.quitaccept
	l.cancel()
	l.shutdown.Wait()
}
