package iorw

import (
	"fmt"
	"io"
)

type ReadWriter interface {
	Reader
	Writer
}

//----------

type Writer interface {
	// insert: Overwrite(i, 0, p)
	// delete: Overwrite(i, n, nil)
	Overwrite(i, n int, p []byte) error
}

//----------

type Reader interface {
	ReadRuneAt(i int) (ru rune, size int, err error)
	ReadLastRuneAt(i int) (ru rune, size int, err error)

	// there must be at least N bytes available or there will be an error
	ReadNAtFast(i, n int) ([]byte, error) // []byte might not be a copy
	ReadNAtCopy(i, n int) ([]byte, error)

	// min>=0 && min<=max && max<=length
	Min() int
	Max() int
}

//----------

var ErrBadIndex = fmt.Errorf("bad index: %w", io.EOF)

func NewErrBadIndex(f string, args ...interface{}) error {
	u := append([]interface{}{ErrBadIndex}, args...)
	return fmt.Errorf("%w: "+f, u...)
}
