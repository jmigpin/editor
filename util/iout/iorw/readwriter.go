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
	Insert(i int, p []byte) error
	Delete(i, n int) error
	Overwrite(i, n int, p []byte) error
}

//----------

type Reader interface {
	ReadRuneAt(i int) (ru rune, size int, err error)
	ReadLastRuneAt(i int) (ru rune, size int, err error)

	// there must be at least N bytes available or there will be an error
	ReadNCopyAt(i, n int) ([]byte, error)
	ReadNSliceAt(i, n int) ([]byte, error) // []byte might not be a copy
	// TODO
	//ReadNAt(i int, p []byte) error // allows allocation outside

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
