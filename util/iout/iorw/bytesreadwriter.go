package iorw

import (
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

type BytesReadWriter struct {
	buf []byte
}

func NewBytesReadWriter(b []byte) *BytesReadWriter {
	return &BytesReadWriter{buf: b}
}

func (rw *BytesReadWriter) Min() int {
	return 0
}
func (rw *BytesReadWriter) Max() int {
	return len(rw.buf)
}

//----------

func (rw *BytesReadWriter) ReadRuneAt(i int) (ru rune, size int, err error) {
	if i < 0 || i > len(rw.buf) {
		return 0, 0, errors.New("bad index")
	}
	ru, size = utf8.DecodeRune(rw.buf[i:])
	if size == 0 {
		return 0, 0, io.EOF
	}
	return ru, size, nil
}

func (rw *BytesReadWriter) ReadLastRuneAt(i int) (ru rune, size int, err error) {
	if i < 0 || i > len(rw.buf) {
		return 0, 0, errors.New("bad index")
	}
	ru, size = utf8.DecodeLastRune(rw.buf[:i])
	if size == 0 {
		return 0, 0, io.EOF
	}
	return ru, size, nil
}

//----------

func (rw *BytesReadWriter) ReadNCopyAt(i, n int) ([]byte, error) {
	b, err := rw.ReadNSliceAt(i, n)
	if err != nil {
		return nil, err
	}
	w := make([]byte, len(b))
	copy(w, b)
	return w, nil
}

func (rw *BytesReadWriter) ReadNSliceAt(i, n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("bad n: %v", n)
	}
	if i < 0 || i > len(rw.buf) {
		return nil, errors.New("bad index")
	}
	if i+n > len(rw.buf) {
		return nil, io.EOF
	}
	return rw.buf[i : i+n], nil
}

//----------

func (rw *BytesReadWriter) Insert(i int, p []byte) error {
	if i < 0 || i > len(rw.buf) {
		return fmt.Errorf("bad index: %v", i)
	}

	l := len(rw.buf)
	rw.buf = append(rw.buf, p...)        // just to increase capacity
	copy(rw.buf[i+len(p):], rw.buf[i:l]) // shift data to the right
	copy(rw.buf[i:], p)                  // insert p

	return nil
}

//----------

func (rw *BytesReadWriter) Delete(i, le int) error {
	if err := rw.delete2(i, le); err != nil {
		return err
	}
	rw.reduceCap()
	return nil
}

func (rw *BytesReadWriter) delete2(i, le int) error {
	if i < 0 || i+le > len(rw.buf) {
		return fmt.Errorf("bad index: %v", i)
	}
	if le == 0 {
		return nil
	}
	if le < 0 {
		return fmt.Errorf("bad len: %v", le)
	}

	copy(rw.buf[i:], rw.buf[i+le:])
	rw.buf = rw.buf[:len(rw.buf)-le]

	return nil
}

// Reduce capacity if too small, to release mem
func (rw *BytesReadWriter) reduceCap() {
	if len(rw.buf) > 1024 && len(rw.buf)*3 < cap(rw.buf) {
		rw.buf = append([]byte{}, rw.buf...)
	}
}

//----------

func (rw *BytesReadWriter) Overwrite(i, length int, p []byte) error {
	if err := rw.delete2(0, length); err != nil {
		return err
	}
	if err := rw.Insert(i, p); err != nil {
		return err
	}
	rw.reduceCap()
	return nil
}

//----------

func NewStringReader(s string) Reader {
	return &BytesReadWriter{buf: []byte(s)}
}
