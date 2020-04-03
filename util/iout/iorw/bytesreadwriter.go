package iorw

import (
	"fmt"
	"io"
)

type BytesReadWriterAt struct {
	buf []byte
}

func NewBytesReadWriterAt(b []byte) *BytesReadWriterAt {
	return &BytesReadWriterAt{b}
}

//----------

// Implement ReaderAt
func (rw *BytesReadWriterAt) ReadFastAt(i, n int) ([]byte, error) {
	if i < 0 {
		return nil, fmt.Errorf("bad index: %v<0", i)
	}
	if i > len(rw.buf) {
		return nil, fmt.Errorf("bad index: %v>%v", i, len(rw.buf))
	}

	// before "i==len" to allow reading an empty buffer (ex: readfull("") without err)
	if n == 0 {
		return nil, nil
	}
	if n < 0 {
		return nil, fmt.Errorf("bad arg: %v<0", n)
	}

	if i == len(rw.buf) {
		return nil, io.EOF
	}

	// i>=0 && i<len && n>=0 -> n>=1
	if i+n > len(rw.buf) {
		n = len(rw.buf) - i
	}

	return rw.buf[i : i+n], nil
}

// Implement ReaderAt
func (rw *BytesReadWriterAt) Min() int { return 0 }

// Implement ReaderAt
func (rw *BytesReadWriterAt) Max() int { return len(rw.buf) }

//----------

// Implement WriterAt
func (rw *BytesReadWriterAt) OverwriteAt(i, del int, p []byte) error {
	// delete
	if i+del > len(rw.buf) {
		return fmt.Errorf("iorw.OverwriteAt: del %v>%v", i+del, len(rw.buf))
	}
	copy(rw.buf[i:], rw.buf[i+del:])
	rw.buf = rw.buf[:len(rw.buf)-del]
	// insert
	l := len(rw.buf)
	if l+len(p) <= cap(rw.buf) {
		rw.buf = rw.buf[:l+len(p)] // increase length
	} else {
		rw.buf = append(rw.buf, p...) // increase capacity
	}
	copy(rw.buf[i+len(p):], rw.buf[i:l]) // shift data to the right
	n := copy(rw.buf[i:], p)
	if n != len(p) {
		return fmt.Errorf("iorw.OverwriteAt: failed full write: %v!=%v", n, len(p))
	}

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
