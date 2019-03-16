package iorw

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode"
)

var ErrLimitReached = errors.New("limit reached")

//----------

func HasPrefix(rs Reader, i int, s []byte) bool {
	if len(s) == 0 {
		return true
	}
	b, err := rs.ReadNSliceAt(i, len(s))
	if err != nil {
		return false
	}
	return bytes.HasPrefix(b, s)
}

//----------

// Returns (-1, nil) if not found.
func Index(r Reader, i, le int, sep []byte, toLower bool) (int, error) {
	return index2(r, i, le, sep, toLower, 32*1024)
}

func index2(r Reader, i, le int, sep []byte, toLower bool, chunk int) (int, error) {
	if chunk < len(sep) {
		return -1, fmt.Errorf("chunk smaller then sep")
	}

	b := i + le
	if b > r.Len() {
		b = r.Len()
	}

	for a := i; a < b; a += chunk - (len(sep) - 1) {
		j := chunk
		if j > b-a {
			j = b - a
		}

		i, err := index3(r, a, j, sep, toLower)
		if err != nil || i >= 0 {
			return i, err
		}
	}

	return -1, nil
}

func index3(r Reader, i, len int, sep []byte, toLower bool) (int, error) {
	p, err := r.ReadNSliceAt(i, len)
	if err != nil {
		return 0, err
	}

	// ignore case
	if toLower {
		p = bytes.ToLower(p)
	}

	j := bytes.Index(p, sep)
	if j >= 0 {
		return i + j, nil
	}
	return -1, nil
}

//----------

func IndexFunc(r Reader, i, len int, truth bool, f func(rune) bool) (index, size int, err error) {
	max := i + len
	for {
		ru, size, err := r.ReadRuneAt(i)
		if err != nil {
			return 0, 0, err
		}
		if i+size > max {
			return i, size, ErrLimitReached
		}
		if f(ru) == truth {
			return i, size, nil
		}
		i += size
	}
}

func LastIndexFunc(r Reader, i, len int, truth bool, f func(rune) bool) (index, size int, err error) {
	min := i - len
	for {
		ru, size, err := r.ReadLastRuneAt(i)
		if err != nil {
			return 0, 0, err
		}
		i -= size
		if i < min {
			return i, size, ErrLimitReached
		}
		if f(ru) == truth {
			return i, size, nil
		}
	}
}

//----------

func LineStartIndex(r Reader, i int) (int, error) {
	k, size, err := NewLineLastIndex(r, i)
	if err == io.EOF {
		return 0, nil
	}
	return k + size, err
}

func LineEndIndex(r Reader, i int) (int, bool, error) {
	k, size, err := NewLineIndex(r, i)
	if err == io.EOF {
		return r.Len(), false, nil
	}
	isNewLine := err == nil
	return k + size, isNewLine, err
}

//----------

var NewLineIndexMax = 2500

func NewLineIndex(r Reader, i int) (int, int, error) {
	newlinef := func(ru rune) bool { return ru == '\n' }
	return IndexFunc(r, i, NewLineIndexMax, true, newlinef)
}

func NewLineLastIndex(r Reader, i int) (int, int, error) {
	newlinef := func(ru rune) bool { return ru == '\n' }
	return LastIndexFunc(r, i, NewLineIndexMax, true, newlinef)
}

//----------

func LinesIndexes(r Reader, a, b int) (int, int, bool, error) {
	ls, err := LineStartIndex(r, a)
	if err != nil {
		return 0, 0, false, err
	}
	le, newline, err := LineEndIndex(r, b)
	if err != nil {
		return 0, 0, false, err
	}
	return ls, le, newline, nil
}

//----------

func IsWordRune(ru rune) bool {
	return unicode.IsLetter(ru) || unicode.IsDigit(ru) || ru == '_' || ru == 0
}

func WordAtIndex(r Reader, index, max int) ([]byte, int, error) {
	// right side
	i1, _, err := IndexFunc(r, index, max, false, IsWordRune)
	if err != nil {
		if err == io.EOF {
			i1 = r.Len()
		} else {
			return nil, 0, err
		}
	}
	if i1 == index { // don't match word at index
		return nil, 0, errors.New("word not found")
	}

	// left side
	i0, size, err := LastIndexFunc(r, index, max, false, IsWordRune)
	if err != nil {
		if err == io.EOF {
			i0 = 0
		} else {
			return nil, 0, err
		}
	} else {
		i0 += size
	}

	s, err := r.ReadNCopyAt(i0, i1-i0)
	if err != nil {
		return nil, 0, err
	}

	return s, i0, nil
}

func WordIsolated(r Reader, i, le int) bool {
	// previous rune can't be a word rune
	ru, _, err := r.ReadLastRuneAt(i)
	if err == nil && IsWordRune(ru) {
		return false
	}
	// next rune can't be a word rune
	ru, _, err = r.ReadRuneAt(i + le)
	if err == nil && IsWordRune(ru) {
		return false
	}
	return true
}
