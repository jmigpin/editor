package iout

import (
	"bufio"
	"io"
	"sync"
	"time"
)

// Flushes after x time if the buffer doesn't get filled. Safe to use concurrently.
type AutoBufWriter struct {
	wc io.WriteCloser
	mu struct {
		sync.Mutex
		buf   *bufio.Writer
		timer *time.Timer
	}
}

func NewAutoBufWriter(wc io.WriteCloser) *AutoBufWriter {
	w := &AutoBufWriter{wc: wc}
	w.mu.buf = bufio.NewWriter(wc)
	return w
}

// Implements io.Closer
func (w *AutoBufWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.clearTimer()
	w.mu.buf.Flush()
	return w.wc.Close()
}

// Implements io.Writer
func (w *AutoBufWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.mu.buf.Write(p)
	w.autoFlush()
	return n, err
}

//----------

func (w *AutoBufWriter) autoFlush() {
	if w.mu.buf.Buffered() == 0 {
		w.clearTimer()
		return
	}
	if w.mu.timer == nil {
		w.mu.timer = time.AfterFunc(50*time.Millisecond, w.flushTime)
	}
}
func (w *AutoBufWriter) flushTime() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.mu.buf.Flush()
	w.clearTimer()
}

func (w *AutoBufWriter) clearTimer() {
	if w.mu.timer != nil {
		w.mu.timer.Stop()
		w.mu.timer = nil
	}
}
