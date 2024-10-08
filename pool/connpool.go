package pool

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/Snawoot/steady-tun/clock"
	clog "github.com/Snawoot/steady-tun/log"
	"github.com/Snawoot/steady-tun/queue"
)

type ConnFactory = func(context.Context) (net.Conn, error)

type ConnPool struct {
	size              uint
	ttl, backoff      time.Duration
	connFactory       ConnFactory
	waiters, prepared *queue.RAQueue
	qmux              sync.Mutex
	logger            *clog.CondLogger
	ctx               context.Context
	cancel            context.CancelFunc
	shutdown          sync.WaitGroup
}

type watchedConn struct {
	conn       net.Conn
	cancel     context.CancelFunc
	canceldone chan struct{}
}

func NewConnPool(size uint, ttl, backoff time.Duration,
	connFactory ConnFactory, logger *clog.CondLogger) *ConnPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &ConnPool{
		size:        size,
		ttl:         ttl,
		backoff:     backoff,
		connFactory: connFactory,
		waiters:     queue.NewRAQueue(),
		prepared:    queue.NewRAQueue(),
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (p *ConnPool) Start() {
	p.shutdown.Add(int(p.size))
	for i := uint(0); i < p.size; i++ {
		go p.worker()
	}
}

func (p *ConnPool) do_backoff() {
	select {
	case <-clock.AfterWallClock(p.backoff):
	case <-p.ctx.Done():
	}
}

func (p *ConnPool) kill_prepared(queue_id uint, watched *watchedConn, output_ch chan *watchedConn) {
	p.qmux.Lock()
	deleted_elem := p.prepared.Delete(queue_id)
	p.qmux.Unlock()
	if deleted_elem == nil {
		// Someone already grabbed this slot from queue. Dispatch anyway.
		p.logger.Debug("Dead conn %v was grabbed from queue", watched.conn.LocalAddr())
		output_ch <- watched
	} else {
		watched.conn.Close()
	}
}

func (p *ConnPool) worker() {
	defer p.shutdown.Done()
	output_ch := make(chan *watchedConn)
	dummybuf := make([]byte, 1)
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
		}
		conn, err := p.connFactory(p.ctx)
		if err != nil {
			select {
			case <-p.ctx.Done():
				return
			default:
				p.logger.Error("Upstream connection error: %v", err)
				p.do_backoff()
				continue
			}
		}
		localaddr := conn.LocalAddr()
		p.logger.Debug("Established upstream connection %v", localaddr)

		p.qmux.Lock()
		waiter := p.waiters.Pop()
		if waiter != nil {
			p.qmux.Unlock()
			waiter.(chan net.Conn) <- conn
			p.logger.Warning("Pool connection delivered directly to waiter")
		} else {
			queue_id := p.prepared.Push(output_ch)
			p.qmux.Unlock()
			readctx, readcancel := context.WithCancel(p.ctx)
			readdone := make(chan struct{}, 1)
			go func() {
				connReadContext(readctx, conn, dummybuf)
				close(readdone)
			}()
			watched := &watchedConn{conn, readcancel, readdone}
			select {
			// Connection delivered via queue
			case output_ch <- watched:
				p.logger.Debug("Pool connection %v delivered via queue", localaddr)
			// Connection disrupted
			case <-readdone:
				p.logger.Debug("Pool connection %v was disrupted", localaddr)
				p.kill_prepared(queue_id, watched, output_ch)
				p.do_backoff()
			// Expired
			case <-clock.AfterWallClock(p.ttl):
				p.logger.Debug("Connection %v seem to be expired", localaddr)
				p.kill_prepared(queue_id, watched, output_ch)
			// Pool context cancelled
			case <-p.ctx.Done():
				conn.Close()
			}
		}
	}
}

func (p *ConnPool) Get(ctx context.Context) chan net.Conn {
	out := make(chan net.Conn, 1)
	p.qmux.Lock()
	free := p.prepared.Pop()
	if free == nil {
		waiter_ch := make(chan net.Conn, 1)
		queue_id := p.waiters.Push(waiter_ch)
		p.qmux.Unlock()
		go func() {
			select {
			case <-ctx.Done():
				out <- nil
				// Try to remove conn request from waiter queue
				p.qmux.Lock()
				p.waiters.Delete(queue_id)
				p.qmux.Unlock()
			case <-p.ctx.Done():
				out <- nil
			case conn := <-waiter_ch:
				out <- conn
			}
		}()
	} else {
		p.qmux.Unlock()
		watched := <-(free.(chan *watchedConn))
		watched.cancel()
		<-watched.canceldone
		out <- watched.conn
	}
	return out
}

func (p *ConnPool) Stop() {
	p.cancel()
	p.shutdown.Wait()
}

func connReadContext(ctx context.Context, conn net.Conn, p []byte) (n int, err error) {
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		n, err = conn.Read(p)
	}()
	select {
	case <-ctx.Done():
		conn.SetReadDeadline(time.Unix(0, 0))
		<-readDone
		conn.SetReadDeadline(time.Time{})
	case <-readDone:
	}
	return
}
