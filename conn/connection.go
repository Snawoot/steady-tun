package conn

import (
	"context"
	"net"
	"time"
)

var ZEROTIME time.Time
var EPOCH = time.Unix(0, 0)

type WrappedConn interface {
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	ReadContext(ctx context.Context, p []byte) (n int, err error)
}

type wrappedConn struct {
	conn net.Conn
}

func (c *wrappedConn) Read(p []byte) (n int, err error) {
	return c.conn.Read(p)
}

func (c *wrappedConn) Write(p []byte) (n int, err error) {
	return c.conn.Write(p)
}

func (c *wrappedConn) Close() error {
	return c.conn.Close()
}

func (c *wrappedConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *wrappedConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *wrappedConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *wrappedConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *wrappedConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

func (c *wrappedConn) ReadContext(ctx context.Context, p []byte) (n int, err error) {
	ch := make(chan struct{}, 1)
	go func() {
		n, err = c.conn.Read(p)
		ch <- struct{}{}
	}()
	select {
	case <-ctx.Done():
		c.conn.SetReadDeadline(EPOCH)
		<-ch
		c.conn.SetReadDeadline(ZEROTIME)
	case <-ch:
	}
	return
}
