package iout

import (
	"io"
	"sync"
)

type SafeWriter struct {
	sync.Mutex
	w io.Writer
}

func NewSafeWriter(w io.Writer) *SafeWriter {
	return &SafeWriter{w: w}
}
func (w *SafeWriter) Write(p []byte) (n int, err error) {
	w.Lock()
	defer w.Unlock()
	return w.w.Write(p)
}
