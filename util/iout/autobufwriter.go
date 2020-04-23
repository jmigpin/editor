package iout

import (
	"bufio"
	"io"
	"sync"
	"time"
)

//godebug:annotatefile

// small amounts of output need to be flushed (not filling the buffer)
const abwUpdatesPerSecond = 10

// Flushes after x time if the buffer doesn't get filled. Safe to use concurrently.
type AutoBufWriter struct {
	w  io.Writer
	mu struct {
		sync.Mutex
		buf   *bufio.Writer
		timer *time.Timer
	}
}

func NewAutoBufWriter(w io.Writer, size int) *AutoBufWriter {
	abw := &AutoBufWriter{w: w}
	abw.mu.buf = bufio.NewWriterSize(abw.w, size)
	return abw
}

// Implements io.Closer
func (w *AutoBufWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.clearTimer()
	w.mu.buf.Flush()
	return nil
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
		return
	}
	if w.mu.timer == nil {
		t := time.Second / abwUpdatesPerSecond
		w.mu.timer = time.AfterFunc(t, w.flushTime)
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
