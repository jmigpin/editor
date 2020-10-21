package iorw

import (
	"fmt"
	"io"
)

type LimitedReaderAt struct {
	ReaderAt
	min, max int
}

// min<=max; allows arguments min<0 && max>length
func NewLimitedReaderAt(r ReaderAt, min, max int) *LimitedReaderAt {
	if min > max {
		//panic(fmt.Sprintf("bad min/max: %v>%v", min, max))
		max = min
	}
	return &LimitedReaderAt{r, min, max}
}

func NewLimitedReaderAtPad(r ReaderAt, min, max, pad int) *LimitedReaderAt {
	return NewLimitedReaderAt(r, min-pad, max+pad)
}

//----------

func (r *LimitedReaderAt) ReadFastAt(i, n int) ([]byte, error) {
	if i < r.min {
		return nil, fmt.Errorf("limited index: %v<%v: %w", i, r.min, io.EOF)
	}
	if i+n > r.max {
		if i > r.max {
			return nil, fmt.Errorf("limited index: %v>%v: %w", i, r.max, io.EOF)
		}
		// n>0, there is data to read
		n = r.max - i
	}
	return r.ReaderAt.ReadFastAt(i, n)
}

func (r *LimitedReaderAt) Min() int {
	u := r.ReaderAt.Min()
	if u > r.min {
		return u
	}
	return r.min
}

func (r *LimitedReaderAt) Max() int {
	u := r.ReaderAt.Max()
	if u < r.max {
		return u
	}
	return r.max
}
