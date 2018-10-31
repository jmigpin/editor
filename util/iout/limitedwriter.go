package iout

import (
	"bytes"
	"fmt"
)

type LimitedWriter struct {
	size int
	buf  bytes.Buffer
}

func NewLimitedWriter(size int) *LimitedWriter {
	return &LimitedWriter{size: size}
}

func (w *LimitedWriter) Write(p []byte) (n int, err error) {
	if w.size < len(p) {
		p = p[:w.size]
		err = fmt.Errorf("limit reached")
	}
	n, err2 := w.buf.Write(p)
	if err2 != nil {
		return n, err2
	}
	w.size -= n
	return n, err
}

func (w *LimitedWriter) Bytes() []byte {
	return w.buf.Bytes()
}
