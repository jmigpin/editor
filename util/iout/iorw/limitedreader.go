package iorw

import (
	"errors"

	"github.com/jmigpin/editor/util/mathutil"
)

var ErrLimitReached = errors.New("limit reached")

// Limits reading while keeping the original offsets.
type LimitedReader struct {
	Reader
	min, max int
}

func NewLimitedReader(r Reader, min, max, pad int) *LimitedReader {
	if min > max {
		panic("min>max")
	}
	if pad < 0 {
		panic("pad<0")
	}
	return &LimitedReader{Reader: r, min: min - pad, max: max + pad}
}

func NewLimitedReaderLen(r Reader, offset, n int) *LimitedReader {
	return NewLimitedReader(r, offset, offset+n, 0)
}

//----------

func (r *LimitedReader) Min() int {
	return mathutil.Biggest(r.min, r.Reader.Min())
}
func (r *LimitedReader) Max() int {
	return mathutil.Smallest(r.max, r.Reader.Max())
}

//----------

func (r *LimitedReader) ReadRuneAt(i int) (ru rune, size int, err error) {
	if i < r.min || i >= r.max {
		return 0, 0, ErrLimitReached
	}
	return r.Reader.ReadRuneAt(i)
}
func (r *LimitedReader) ReadLastRuneAt(i int) (ru rune, size int, err error) {
	if i <= r.min || i > r.max {
		return 0, 0, ErrLimitReached
	}
	return r.Reader.ReadLastRuneAt(i)
}
func (r *LimitedReader) ReadNCopyAt(i, n int) ([]byte, error) {
	if i < r.min || i+n > r.max {
		return nil, ErrLimitReached
	}
	return r.Reader.ReadNCopyAt(i, n)
}
func (r *LimitedReader) ReadNSliceAt(i, n int) ([]byte, error) {
	if i < r.min || i+n > r.max {
		return nil, ErrLimitReached
	}
	return r.Reader.ReadNSliceAt(i, n)
}
