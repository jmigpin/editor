package iorw

import (
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

// TODO: rename BytesReadWriter or BytesRW
type RW struct {
	buf []byte
}

func NewRW(b []byte) *RW {
	return &RW{buf: b}
}

func (rw *RW) Len() int {
	return len(rw.buf)
}

//----------

func (rw *RW) ReadRuneAt(i int) (ru rune, size int, err error) {
	if i < 0 || i > len(rw.buf) {
		return 0, 0, errors.New("bad index")
	}
	ru, size = utf8.DecodeRune(rw.buf[i:])
	if size == 0 {
		return 0, 0, io.EOF
	}
	return ru, size, nil
}

func (rw *RW) ReadLastRuneAt(i int) (ru rune, size int, err error) {
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

func (rw *RW) ReadNAt(i, n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("bad n: %v", n)
	}
	if i < 0 || i > len(rw.buf) {
		return nil, errors.New("bad index")
	}
	if i+n > len(rw.buf) {
		return nil, io.EOF
	}
	w := make([]byte, n)
	copy(w, rw.buf[i:i+n])
	return w, nil
}

func (rw *RW) ReadNSliceAt(i, n int) ([]byte, error) {
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

//func (rw *RW) ReadAtMost(i, n int) ([]byte, error) {
//	if n < 0 {
//		return nil, fmt.Errorf("bad n: %v", n)
//	}
//	if i < 0 || i > len(rw.buf) {
//		return nil, errors.New("bad index")
//	}
//	b := rw.buf[i:]
//	if len(b) < n {
//		n = len(b)
//	}
//	w := make([]byte, n)
//	copy(w, b[:n])
//	return w, nil
//}

//----------

//----------

func (rw *RW) Insert(i int, p []byte) error {
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

func (rw *RW) Delete(i, le int) error {
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

//----------

//func (rw *RW) Overwrite(i, le int, p []byte) error {
//	if i < 0 || i+le > len(rw.buf) {
//		return errors.New("bad index")
//	}
//	lp := len(p)
//	if le < lp {
//		copy(rw.buf[i:], p[:le]) // overwrite
//		return rw.Insert(i+le, p[le:])
//	} else {
//		// delete
//		if err := rw.Delete(i+lp, le-lp); err != nil {
//			return err
//		}
//		// overwrite
//		copy(rw.buf[i:], p)
//		return nil
//	}
//}
