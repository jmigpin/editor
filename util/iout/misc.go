package iout

import "bytes"

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
