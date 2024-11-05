package dnscache

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/jellydator/ttlcache/v3"
)

type ContextDialer = func(ctx context.Context, network, address string) (net.Conn, error)

type cacheKey struct {
	network string
	host    string
}

type cacheValue struct {
	addrs []netip.Addr
	err   error
}

type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

func WrapDialer(dialer ContextDialer, resolver Resolver, size int, posTTL, negTTL, timeout time.Duration) ContextDialer {
	cache := ttlcache.New[cacheKey, cacheValue](
		ttlcache.WithDisableTouchOnHit[cacheKey, cacheValue](),
		ttlcache.WithLoader(
			ttlcache.NewSuppressedLoader(
				ttlcache.LoaderFunc[cacheKey, cacheValue](
					func(c *ttlcache.Cache[cacheKey, cacheValue], key cacheKey) *ttlcache.Item[cacheKey, cacheValue] {
						ctx, cl := context.WithTimeout(context.Background(), timeout)
						defer cl()
						res, err := resolver.LookupNetIP(ctx, key.network, key.host)
						setTTL := negTTL
						if err == nil {
							setTTL = posTTL
						}
						return c.Set(key, cacheValue{
							addrs: res,
							err:   err,
						}, setTTL)
					},
				),
				nil),
		),
		ttlcache.WithCapacity[cacheKey, cacheValue](uint64(size)),
	)
	wrapped := func(ctx context.Context, network, address string) (net.Conn, error) {
		var resolveNetwork string
		switch network {
		case "udp4", "tcp4", "ip4":
			resolveNetwork = "ip4"
		case "udp6", "tcp6", "ip6":
			resolveNetwork = "ip6"
		case "udp", "tcp", "ip":
			resolveNetwork = "ip"
		default:
			return nil, fmt.Errorf("resolving dial %q: unsupported network %q", address, network)
		}

		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("failed to extract host and port from %s: %w", address, err)
		}

		resItem := cache.Get(cacheKey{
			network: resolveNetwork,
			host: host,
		})
		if resItem == nil {
			return nil, fmt.Errorf("cache lookup failed for pair <%q, %q>", resolveNetwork, host)
		}

		res := resItem.Value()
		if res.err != nil {
			return nil, res.err
		}

		var conn net.Conn

		for _, ip := range res.addrs {
			conn, err = dialer(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
		}

		return nil, fmt.Errorf("failed to dial %s: %w", address, err)
	}
	return wrapped
}
