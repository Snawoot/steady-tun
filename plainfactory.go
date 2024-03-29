package main

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"
)

type PlainConnFactory struct {
	addr   string
	dialer *net.Dialer
	logger *CondLogger
}

func NewPlainConnFactory(host string, port uint16, timeout time.Duration, logger *CondLogger) *PlainConnFactory {
	return &PlainConnFactory{
		addr:   net.JoinHostPort(host, strconv.Itoa(int(port))),
		dialer: &net.Dialer{Timeout: timeout},
		logger: logger,
	}
}

func (cf *PlainConnFactory) DialContext(ctx context.Context) (WrappedConn, error) {
	conn, err := cf.dialer.DialContext(ctx, "tcp", cf.addr)
	if err != nil {
		return nil, fmt.Errorf("cf.dialer.DialContext(ctx, \"tcp\", %q) failed: %v", cf.addr, err)
	}
	return &wrappedConn{
		conn:   conn,
		logger: cf.logger,
	}, nil
}
