package iorw

import "io"

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

	Min() int
	Max() int
}

//----------

// min/max length
func MMLen(rd Reader) int {
	return rd.Max() - rd.Min()
}

// Returns a slice (not a copy).
func ReadFullSlice(rd Reader) ([]byte, error) {
	min, max := rd.Min(), rd.Max()
	return rd.ReadNSliceAt(min, max-min)
}

func SetString(rw ReadWriter, s string) error {
	min, max := rw.Min(), rw.Max()
	return rw.Overwrite(min, max-min, []byte(s))
}

//----------

// Iterate over n+1 runes, with the last rune being eofRune(-1).
func ReaderIter(r Reader, fn func(i int, ru rune) bool) error {
	o := r.Min()
	n := r.Max() - r.Min()
	for i := o; ; {
		if i >= o+n {
			_ = fn(i, EndRune)
			return nil
		}
		ru, size, err := r.ReadRuneAt(i)
		if err != nil {
			if err == io.EOF {
				_ = fn(i, EndRune)
				return nil
			}
			return err
		}
		if !fn(i, ru) {
			break
		}
		i += size
	}
	return nil
}

const EndRune = -1
