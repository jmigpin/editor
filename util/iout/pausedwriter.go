package iout

import (
	"io"
	"sync"
)

type PausedWriter struct {
	Wr     io.Writer
	mu     sync.Mutex
	cond   *sync.Cond
	paused bool
}

func NewPausedWriter(wr io.Writer) *PausedWriter {
	pw := &PausedWriter{Wr: wr}
	pw.cond = sync.NewCond(&pw.mu)
	pw.paused = true
	return pw
}
func (pw *PausedWriter) Write(b []byte) (int, error) {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	for pw.paused {
		pw.cond.Wait()
	}
	return pw.Wr.Write(b)
}
func (pw *PausedWriter) Pause(v bool) {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	pw.paused = v
	pw.cond.Signal()
}
func (pw *PausedWriter) Unpause() {
	pw.Pause(false)
}
