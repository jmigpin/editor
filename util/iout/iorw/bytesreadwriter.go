package iorw

import (
	"io"
	"unicode/utf8"
)

type BytesReadWriter struct {
	buf []byte
}

func NewBytesReadWriter(b []byte) *BytesReadWriter {
	return &BytesReadWriter{buf: b}
}

//----------

func (rw *BytesReadWriter) Min() int {
	return 0
}
func (rw *BytesReadWriter) Max() int {
	return len(rw.buf)
}

//----------

func (rw *BytesReadWriter) ReadRuneAt(i int) (ru rune, size int, err error) {
	if err := checkIndex(0, len(rw.buf), i); err != nil {
		return 0, 0, err
	}
	ru, size = utf8.DecodeRune(rw.buf[i:])
	if size == 0 {
		return 0, 0, io.EOF
	}
	return ru, size, nil
}

func (rw *BytesReadWriter) ReadLastRuneAt(i int) (ru rune, size int, err error) {
	if err := checkIndex(0, len(rw.buf), i); err != nil {
		return 0, 0, err
	}
	ru, size = utf8.DecodeLastRune(rw.buf[:i])
	if size == 0 {
		return 0, 0, io.EOF
	}
	return ru, size, nil
}

//----------

func (rw *BytesReadWriter) ReadNAtFast(i, n int) ([]byte, error) {
	if err := checkIndexN(0, len(rw.buf), i, n); err != nil {
		return nil, err
	}
	return rw.buf[i : i+n], nil
}

func (rw *BytesReadWriter) ReadNAtCopy(i, n int) ([]byte, error) {
	b, err := rw.ReadNAtFast(i, n)
	if err != nil {
		return nil, err
	}
	p := make([]byte, n)
	copy(p, b)
	return p, nil
}

//----------

func (rw *BytesReadWriter) Overwrite(i, n int, p []byte) error {
	if err := checkIndexN(0, len(rw.buf), i, n); err != nil {
		return err
	}
	// delete
	copy(rw.buf[i:], rw.buf[i+n:])
	rw.buf = rw.buf[:len(rw.buf)-n]
	// insert
	l := len(rw.buf)
	rw.buf = append(rw.buf, p...)        // just to increase capacity
	copy(rw.buf[i+len(p):], rw.buf[i:l]) // shift data to the right
	copy(rw.buf[i:], p)                  // insert p

	rw.buf = autoReduceCap(rw.buf)
	return nil
}

//----------

func autoReduceCap(p []byte) []byte {
	if len(p) > 1024 && len(p) < 3*cap(p) {
		return append([]byte{}, p...)
	}
	return p
}

//----------

func checkIndex(min, max, i int) error {
	if i < min {
		return NewErrBadIndex("%v, min=%v", i, min)
	}
	if i > max { // allow max
		return NewErrBadIndex("%v, max=%v", i, max)
	}
	return nil
}

func checkIndexN(min, max, i, n int) error {
	if n < 0 {
		return NewErrBadIndex("n=%v", n)
	}
	if i < min {
		return NewErrBadIndex("%v, min=%v", i, min)
	}
	if i+n > max {
		return NewErrBadIndex("i+n=%v, max=%v", i+n, max)
	}
	return nil
}
