package conn

import (
	"context"
	"net"
)

type ContextDialer = func(ctx context.Context, network, address string) (net.Conn, error)

type Factory interface {
	DialContext(ctx context.Context) (net.Conn, error)
}
