package iorw

import "fmt"

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
		panic(fmt.Sprintf("min>max: %v>%v", min, max))
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

//----------

func (r *LimitedReader) ReadRuneAt(i int) (ru rune, size int, err error) {
	if err := checkIndex(r.Min(), r.Max(), i); err != nil {
		return 0, 0, err
	}
	return r.Reader.ReadRuneAt(i)
}
func (r *LimitedReader) ReadLastRuneAt(i int) (ru rune, size int, err error) {
	if err := checkIndex(r.Min(), r.Max(), i); err != nil {
		return 0, 0, err
	}
	return r.Reader.ReadLastRuneAt(i)
}
func (r *LimitedReader) ReadNAtFast(i, n int) ([]byte, error) {
	if err := checkIndexN(r.Min(), r.Max(), i, n); err != nil {
		return nil, err
	}
	return r.Reader.ReadNAtFast(i, n)
}
func (r *LimitedReader) ReadNAtCopy(i, n int) ([]byte, error) {
	if err := checkIndexN(r.Min(), r.Max(), i, n); err != nil {
		return nil, err
	}
	return r.Reader.ReadNAtCopy(i, n)
}
