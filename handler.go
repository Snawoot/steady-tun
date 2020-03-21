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

func (h *ConnHandler) proxy(left, right net.Conn) {
    wg := sync.WaitGroup{}
    cpy := func (dst, src net.Conn) {
        defer wg.Done()
        b, err := io.Copy(dst, src)
        h.logger.Debug("cpy done: bytes=%d err=%v", b, err)
    }
    wg.Add(2)
    go cpy(left, right)
    go cpy(right, left)
    wg.Wait()
}

func (h *ConnHandler) handle (c net.Conn) {
    remote_addr := c.RemoteAddr()
    defer h.logger.Info("Connection %s done", remote_addr)
    h.logger.Info("Got new connection from %s", remote_addr)
    ctx := context.Background()
    tlsconn, err := h.connfactory.DialContext(ctx)
    if err != nil {
        h.logger.Error("Upstream connection failed: %s", err.Error())
        return
    }
    defer tlsconn.Close()
    h.proxy(c, tlsconn)
}
