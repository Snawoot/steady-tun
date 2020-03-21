package main

import (
    "io"
    "errors"
    "time"
)

const MAX_LOG_QLEN = 128
const QUEUE_SHUTDOWN_TIMEOUT = 500 * time.Millisecond

type LogWriter struct {
    writer io.Writer
    ch chan []byte
    done chan struct{}
}

func (lw *LogWriter) Write(p []byte) (int, error) {
    buf := make([]byte, len(p))
    copy(buf, p)
    select {
    case lw.ch <- buf:
        return len(p), nil
    default:
        return 0, errors.New("Writer queue overflow")
    }
}

func NewLogWriter(writer io.Writer) *LogWriter {
    lw := &LogWriter{writer,
                     make(chan []byte, MAX_LOG_QLEN),
                     make(chan struct{})}
    go lw.loop()
    return lw
}

func (lw *LogWriter) loop() {
    for p := range lw.ch {
        lw.writer.Write(p)
    }
    lw.done <- struct{}{}
}

func (lw *LogWriter) Close() {
    close(lw.ch)
    timer := time.After(QUEUE_SHUTDOWN_TIMEOUT)
    select {
        case <-timer:
        case <-lw.done:
    }
}
