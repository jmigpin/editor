package iorw

import "errors"

var LimitErr = errors.New("out of limits")

// Limits reading while keeps original offsets.
type LimitedReader struct {
	Reader
	min, max int
}

func NewLimitedReader(r Reader, offset, n int) *LimitedReader {
	return &LimitedReader{Reader: r, min: offset, max: offset + n}
}

func (r *LimitedReader) ReadRuneAt(i int) (ru rune, size int, err error) {
	if i < r.min || i >= r.max {
		return 0, 0, LimitErr
	}
	return r.Reader.ReadRuneAt(i)
}
func (r *LimitedReader) ReadLastRuneAt(i int) (ru rune, size int, err error) {
	if i <= r.min || i > r.max {
		return 0, 0, LimitErr
	}
	return r.Reader.ReadLastRuneAt(i)
}
func (r *LimitedReader) ReadNCopyAt(i, n int) ([]byte, error) {
	if i < r.min || i+n > r.max {
		return nil, LimitErr
	}
	return r.Reader.ReadNCopyAt(i, n)
}
func (r *LimitedReader) ReadNSliceAt(i, n int) ([]byte, error) {
	if i < r.min || i+n > r.max {
		return nil, LimitErr
	}
	return r.Reader.ReadNSliceAt(i, n)
}
