package iout

import (
	"bytes"
	"fmt"
)

type LimitedWriter struct {
	avail int
	buf   bytes.Buffer
}

func NewLimitedWriter(size int) *LimitedWriter {
	return &LimitedWriter{avail: size}
}

func (w *LimitedWriter) Write(p []byte) (n int, err error) {
	if w.avail < len(p) {
		p = p[:w.avail]
		err = fmt.Errorf("limit reached")
	}
	n, err2 := w.buf.Write(p)
	if err2 != nil {
		return n, err2
	}
	w.avail -= n
	return n, err
}

func (w *LimitedWriter) Bytes() []byte {
	return w.buf.Bytes()
}
