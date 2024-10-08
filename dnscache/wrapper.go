package dnscache

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/Vonage/gosrvlib/pkg/dnscache"
)

type ContextDialer = func(ctx context.Context, network, address string) (net.Conn, error)

func WrapDialer(dialer ContextDialer, size int, ttl time.Duration) ContextDialer {
	cache := dnscache.New(net.DefaultResolver, size, ttl)
	wrapped := func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("failed to extract host and port from %s: %w", address, err)
		}

		ips, err := cache.LookupHost(ctx, host)
		if err != nil {
			return nil, err
		}

		var conn net.Conn

		for _, ip := range ips {
			conn, err = dialer(ctx, network, net.JoinHostPort(ip, port))
			if err == nil {
				return conn, nil
			}
		}

		return nil, fmt.Errorf("failed to dial %s: %w", address, err)
	}
	return wrapped
}
