package main

import (
    "sync"
    "io"
    "net"
    "time"
    "context"
)

type ConnHandler struct {
    pool *ConnPool
    pool_wait time.Duration
    logger *CondLogger
}

func NewConnHandler(pool *ConnPool, pool_wait time.Duration, logger *CondLogger) *ConnHandler {
    return &ConnHandler{pool, pool_wait, logger}
}

func (h *ConnHandler) proxy(ctx context.Context, left, right net.Conn) {
    wg := sync.WaitGroup{}
    cpy := func (dst, src net.Conn) {
        defer wg.Done()
        b, err := io.Copy(dst, src)
        dst.Close()
        h.logger.Debug("cpy done: bytes=%d err=%v", b, err)
    }
    wg.Add(2)
    go cpy(left, right)
    go cpy(right, left)
    groupdone := make(chan struct{})
    go func() {
        wg.Wait()
        groupdone <-struct{}{}
    }()
    select {
    case <-ctx.Done():
        left.Close()
        right.Close()
    case <-groupdone:
        return
    }
    <-groupdone
    return
}

func (h *ConnHandler) handle (ctx context.Context, c net.Conn) {
    remote_addr := c.RemoteAddr()
    h.logger.Info("Got new connection from %s", remote_addr)

    select {
    case <-AfterWallClock(h.pool_wait):
        h.logger.Error("Timeout while waiting connection from pool")
        c.Close()
    case tlsconn := <-h.pool.Get(ctx):
        if tlsconn == nil {
            select {
            case <-ctx.Done():
            default:
                h.logger.Error("Error on connection retrieve from pool")
            }
            c.Close()
        } else {
            h.proxy(ctx, c, tlsconn)
        }
    }
    h.logger.Info("Connection %s done", remote_addr)
}
