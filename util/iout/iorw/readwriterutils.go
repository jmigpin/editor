package iorw

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"unicode"
)

//----------

func HasPrefix(r Reader, i int, s []byte) bool {
	if len(s) == 0 {
		return true
	}
	b, err := r.ReadNSliceAt(i, len(s))
	if err != nil {
		return false
	}
	return bytes.HasPrefix(b, s)
}

//----------

func Index(r Reader, i int, sep []byte, toLower bool) (int, error) {
	ctx := context.Background()
	return IndexCtx(ctx, r, i, sep, toLower)
}

// Returns (-1, nil) if not found.
func IndexCtx(ctx context.Context, r Reader, i int, sep []byte, toLower bool) (int, error) {
	return indexCtx2(ctx, r, i, sep, toLower, 32*1024)
}

func indexCtx2(ctx context.Context, r Reader, i int, sep []byte, toLower bool, chunk int) (int, error) {
	if chunk < len(sep) {
		return -1, fmt.Errorf("chunk smaller then sep")
	}

	b := r.Len()
	for a := i; a < b; a += chunk - (len(sep) - 1) {
		c := chunk
		if c > b-a {
			c = b - a
		}

		i, err := indexCtx3(r, a, c, sep, toLower)
		if err != nil || i >= 0 {
			return i, err
		}

		if err := ctx.Err(); err != nil {
			return 0, err
		}
	}

	return -1, nil
}

func indexCtx3(r Reader, i, length int, sep []byte, toLower bool) (int, error) {
	p, err := r.ReadNSliceAt(i, length)
	if err != nil {
		return 0, err
	}

	// TODO: ignore accents?
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

func IndexFunc(r Reader, i int, truth bool, f func(rune) bool) (index, size int, err error) {
	for {
		ru, size, err := r.ReadRuneAt(i)
		if err != nil {
			return i, 0, err
		}
		if f(ru) == truth {
			return i, size, nil
		}
		i += size
	}
}

func LastIndexFunc(r Reader, i int, truth bool, f func(rune) bool) (index, size int, err error) {
	for {
		ru, size, err := r.ReadLastRuneAt(i)
		if err != nil {
			return i, 0, err
		}
		i -= size
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

func NewLineIndex(r Reader, i int) (int, int, error) {
	newlinef := func(ru rune) bool { return ru == '\n' }
	return IndexFunc(r, i, true, newlinef)
}

func NewLineLastIndex(r Reader, i int) (int, int, error) {
	newlinef := func(ru rune) bool { return ru == '\n' }
	return LastIndexFunc(r, i, true, newlinef)
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

func WordAtIndex(r Reader, index int) ([]byte, int, error) {
	// right side
	i1, _, err := IndexFunc(r, index, false, IsWordRune)
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
	i0, size, err := LastIndexFunc(r, index, false, IsWordRune)
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
