package iout

import (
	"bufio"
	"io"
	"sync"
	"time"
)

// Auto flushes after x time if the buffer doesn't get filled. Safe to use concurrently.
type AutoBufWriter struct {
	mu    sync.Mutex
	buf   *bufio.Writer
	wc    io.WriteCloser
	timer *time.Timer
}

func NewAutoBufWriter(wc io.WriteCloser) *AutoBufWriter {
	buf := bufio.NewWriter(wc)
	return &AutoBufWriter{buf: buf, wc: wc}
}

// Implements io.Closer
func (w *AutoBufWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.clearTimer()
	w.buf.Flush()
	return w.wc.Close()
}

// Implements io.Writer
func (w *AutoBufWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	defer w.autoFlush() // deferred to run after the write
	return w.buf.Write(p)
}

func (w *AutoBufWriter) autoFlush() {
	if w.buf.Buffered() == 0 {
		w.clearTimer()
		return
	}
	if w.timer == nil {
		w.timer = time.AfterFunc(50*time.Millisecond, w.flushTime)
	}
}
func (w *AutoBufWriter) flushTime() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf.Flush()
	w.clearTimer()
}

func (w *AutoBufWriter) clearTimer() {
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
}
