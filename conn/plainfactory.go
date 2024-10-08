package conn

import (
	"context"
	"fmt"
	"net"
	"strconv"
)

type PlainConnFactory struct {
	addr   string
	dialer ContextDialer
}

var _ Factory = &PlainConnFactory{}

func NewPlainConnFactory(host string, port uint16, dialer ContextDialer) *PlainConnFactory {
	return &PlainConnFactory{
		addr:   net.JoinHostPort(host, strconv.Itoa(int(port))),
		dialer: dialer,
	}
}

func (cf *PlainConnFactory) DialContext(ctx context.Context) (net.Conn, error) {
	conn, err := cf.dialer(ctx, "tcp", cf.addr)
	if err != nil {
		return nil, fmt.Errorf("cf.dialer.DialContext(ctx, \"tcp\", %q) failed: %v", cf.addr, err)
	}
	return conn, nil
}
