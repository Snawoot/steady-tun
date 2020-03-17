package main

import (
    "net"
)

type ConnHandler struct {
    logger *CondLogger
    connfactory *ConnFactory
}

func NewConnHandler(logger *CondLogger, connfactory *ConnFactory) *ConnHandler {
    return &ConnHandler{logger, connfactory}
}

func (h *ConnHandler) handle (c *net.TCPConn) {
    remote_addr := c.RemoteAddr()
    defer h.logger.Info("Connection %s closed", remote_addr)
    h.logger.Info("Got new connection from %s", remote_addr)
    c.Write([]byte("OK\n"))
}
