package main

import (
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
    done := make(chan bool, 2)
    cpy := func (dst, src net.Conn) {
        b, err := io.Copy(dst, src)
        h.logger.Debug("cpy done: bytes=%d err=%v", b, err)
        done <- true
    }
    go cpy(left, right)
    go cpy(right, left)
    <-done
}

func (h *ConnHandler) handle (c net.Conn) {
    remote_addr := c.RemoteAddr()
    defer h.logger.Info("Connection %s closed", remote_addr)
    h.logger.Info("Got new connection from %s", remote_addr)
    ctx := context.Background()
    tlsconn, err := h.connfactory.DialContext(ctx)
    if err != nil {
        h.logger.Error("Upstream connection failed: %s", err.Error())
        return
    }
    defer tlsconn.conn.Close()
    h.proxy(c, tlsconn.conn)
}
