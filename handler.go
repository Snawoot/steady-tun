package main

import (
    "sync"
    "io"
    "net"
    "context"
)

type ConnHandler struct {
    logger *CondLogger
    connfactory *ConnFactory
}

func NewConnHandler(logger *CondLogger, connfactory *ConnFactory) *ConnHandler {
    return &ConnHandler{logger, connfactory}
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

    tlsconn, err := h.connfactory.DialContext(ctx)
    if err != nil {
        h.logger.Error("Upstream connection failed: %s", err.Error())
        c.Close()
        return
    }
    h.proxy(ctx, c, tlsconn)
    h.logger.Info("Connection %s done", remote_addr)
}
