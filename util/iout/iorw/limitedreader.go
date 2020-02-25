package iorw

import (
	"io"
)

//var ErrLimitReached = fmt.Printf("limit reached: %w", io.EOF)
var errLimitReached = io.EOF

// Limits reading while keeping the original offsets.
type LimitedReader struct {
	Reader
	min int
	max int
}

// min<=max; allows arguments min<0 && max>length
func NewLimitedReader(r Reader, min, max int) *LimitedReader {
	if min < 0 {
		min = 0
	}
	if min > max {
		panic("min>max")
	}
	return &LimitedReader{Reader: r, min: min, max: max}
}

func NewLimitedReaderPad(r Reader, min, max, pad int) *LimitedReader {
	return NewLimitedReader(r, min-pad, max+pad)
}

//----------

func (r *LimitedReader) Min() int {
	u := r.Reader.Min()
	if u < r.min {
		u = r.min // upper limitation
	}
	max := r.Max()
	if u > max {
		u = max // lower limitation
	}
	return u

}

func (r *LimitedReader) Max() int {
	u := r.Reader.Max()
	if u > r.max {
		u = r.max // lower limitation
	}
	return u
}

func (r *LimitedReader) indexInBounds(i int) bool {
	return i >= r.Min() && i < r.Max()
}

//----------

func (r *LimitedReader) ReadRuneAt(i int) (ru rune, size int, err error) {
	if !r.indexInBounds(i) {
		return 0, 0, errLimitReached
	}
	return r.Reader.ReadRuneAt(i)
}
func (r *LimitedReader) ReadLastRuneAt(i int) (ru rune, size int, err error) {
	if !r.indexInBounds(i) {
		return 0, 0, errLimitReached
	}
	return r.Reader.ReadLastRuneAt(i)
}
func (r *LimitedReader) ReadNCopyAt(i, n int) ([]byte, error) {
	if !r.indexInBounds(i) {
		return nil, errLimitReached
	}
	return r.Reader.ReadNCopyAt(i, n)
}
func (r *LimitedReader) ReadNSliceAt(i, n int) ([]byte, error) {
	if !r.indexInBounds(i) {
		return nil, errLimitReached
	}
	return r.Reader.ReadNSliceAt(i, n)
}
