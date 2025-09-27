package iout

import (
	"bytes"
	"io"
	"sync"
)

type FnWriter func([]byte) (int, error)

func (w FnWriter) Write(p []byte) (int, error) {
	return w(p)
}

//----------

type FnReader func([]byte) (int, error)

func (r FnReader) Read(p []byte) (int, error) {
	return r(p)
}

//----------

type FnCloser func() error

func (c FnCloser) Close() error {
	return c()
}

//----------

// useful to help instantiate an io.ReadWriteCloser
type RWC struct {
	io.Reader
	io.Writer
	io.Closer
}

//----------

// serializes concurrent writes
type SafeWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func NewSafeWriter(w io.Writer) *SafeWriter {
	return &SafeWriter{w: w}
}
func (s *SafeWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}

//----------
//----------
//----------

func CopyBytes(b []byte) []byte {
	p := make([]byte, len(b), len(b))
	copy(p, b)
	return p
}

func CountLines(b []byte) int {
	return bytes.Count(b, []byte("\n"))
}
