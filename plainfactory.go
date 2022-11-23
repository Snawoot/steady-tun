package main

import (
	"context"
	"fmt"
	"net"
)

type PlainConnFactory struct {
	addr   string
	dialer *net.Dialer
	logger *CondLogger
}

func (cf *PlainConnFactory) DialContext(ctx context.Context) (WrappedConn, error) {
	conn, err := cf.dialer.DialContext(ctx, "tcp", cf.addr)
	if err != nil {
		return nil, fmt.Errorf("cf.dialer.DialContext(ctx, \"tcp\", %q) failed: %v", err)
	}
	return &wrappedConn{
		conn:   conn,
		logger: cf.logger,
	}, nil
}
