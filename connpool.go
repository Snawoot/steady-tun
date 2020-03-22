package main

import (
    "time"
    "context"
    "sync"
    "net"
)

type ConnPool struct {
    size uint
    ttl, backoff time.Duration
    connfactory *ConnFactory
    waiters, prepared *RAQueue
    qmux sync.Mutex
    logger *CondLogger
    ctx context.Context
    cancel context.CancelFunc
    shutdown sync.WaitGroup
}

func NewConnPool(size uint, ttl, backoff time.Duration,
                 connfactory *ConnFactory, logger *CondLogger) *ConnPool {
    ctx, cancel := context.WithCancel(context.Background())
    return &ConnPool{
        size: size,
        ttl: ttl,
        backoff: backoff,
        connfactory: connfactory,
        waiters: NewRAQueue(),
        prepared: NewRAQueue(),
        logger: logger,
        ctx: ctx,
        cancel: cancel,
    }
}

func (p *ConnPool) Start() {
    p.shutdown.Add(int(p.size))
    for i:=uint(0); i<p.size; i++ {
        go p.worker()
    }
}

func (p *ConnPool) do_backoff() {
    select {
    case <-time.After(p.backoff):
    case <-p.ctx.Done():
    }
}

func (p *ConnPool) worker() {
    defer p.shutdown.Done()
    output_ch := make(chan net.Conn)
    for {
        select {
        case <-p.ctx.Done():
            return
        default:
        }
        conn, err := p.connfactory.DialContext(p.ctx)
        if err != nil {
            select {
            case <-p.ctx.Done():
                return
            default:
                p.logger.Error("Upstream connection error: %s", err)
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
            waiter.(chan net.Conn) <-conn
            p.logger.Warning("Pool connection delivered directly to waiter")
        } else {
            queue_id := p.prepared.Push(output_ch)
            p.qmux.Unlock()
            select {
            case output_ch <-conn:
                p.logger.Debug("Pool connection %v delivered via queue", localaddr)
            case <-time.After(p.ttl):
                p.logger.Debug("Connection %v seem to be expired", localaddr)
                p.qmux.Lock()
                deleted_elem := p.prepared.Delete(queue_id)
                p.qmux.Unlock()
                if deleted_elem == nil {
                    // Someone already grabbed this slot from queue. Dispatch anyway.
                    p.logger.Debug("Dead conn %v was grabbed from queue", localaddr)
                    output_ch <-conn
                } else {
                    conn.Close()
                }
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
        go func () {
            select {
            case <-ctx.Done():
                out <-nil
                // Try to remove conn request from waiter queue
                p.qmux.Lock()
                p.waiters.Delete(queue_id)
                p.qmux.Unlock()
            case <-p.ctx.Done():
                out <-nil
            case conn := <-waiter_ch:
                out <-conn
            }
        }()
    } else {
        p.qmux.Unlock()
        out <-<-free.(chan net.Conn)
    }
    return out
}

func (p *ConnPool) Stop() {
    p.cancel()
    p.shutdown.Wait()
}
