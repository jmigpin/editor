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

func (rw *BytesReadWriter) Len() int {
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

	n := len(rw.buf) + len(p)
	if n > cap(rw.buf) {
		// grow capacity
		w := make([]byte, n, n+64)
		copy(w, rw.buf[:i])
		copy(w[i+len(p):], rw.buf[i:])
		copy(w[i:], p)
		rw.buf = w
	} else {
		rw.buf = rw.buf[0:n]
		copy(rw.buf[i+len(p):], rw.buf[i:])
		copy(rw.buf[i:], p)
	}
	return nil
}

func (rw *BytesReadWriter) Delete(i, le int) error {
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

	// reduce capacity if too small, to release mem
	if len(rw.buf) > 2*1024 && len(rw.buf)*2 < cap(rw.buf) {
		if len(rw.buf) == 0 {
			// don't do anything, probably followed by an insert
		} else {
			n := len(rw.buf)
			w := make([]byte, n, n+64)
			copy(w, rw.buf)
			rw.buf = w
		}
	}
	return nil
}
