package server

import (
	"context"
	"io"
	"net"
	"sync"

	clog "github.com/Snawoot/steady-tun/log"
	"github.com/Snawoot/steady-tun/pool"
)

type ConnHandler struct {
	pool   *pool.ConnPool
	logger *clog.CondLogger
}

func NewConnHandler(pool *pool.ConnPool, logger *clog.CondLogger) *ConnHandler {
	return &ConnHandler{pool, logger}
}

func (h *ConnHandler) proxy(ctx context.Context, left, right net.Conn) {
	wg := sync.WaitGroup{}
	cpy := func(dst, src net.Conn) {
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
		groupdone <- struct{}{}
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

func (h *ConnHandler) Handle(ctx context.Context, c net.Conn) {
	remote_addr := c.RemoteAddr()
	h.logger.Info("Got new connection from %s", remote_addr)
	defer h.logger.Info("Connection %s done", remote_addr)

	tlsconn, err := h.pool.Get(ctx)
	if err != nil {
		h.logger.Error("Error on connection retrieve from pool: %v", err)
		c.Close()
	}
	h.proxy(ctx, c, tlsconn)
}
