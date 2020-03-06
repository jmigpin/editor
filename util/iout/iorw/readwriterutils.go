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

func NewStringReader(s string) Reader {
	return &BytesReadWriter{buf: []byte(s)}
}

//----------

// min/max length
func RLen(rd Reader) int {
	return rd.Max() - rd.Min()
}

func REqual(rd Reader, b []byte) (bool, error) {
	u, err := ReadFullFast(rd)
	if err != nil {
		return false, err
	}
	if len(u) != len(b) {
		return false, nil
	}
	return bytes.Equal(u, b), nil
}

//----------

// Result might not be a copy.
func ReadFullFast(rd Reader) ([]byte, error) {
	min, max := rd.Min(), rd.Max()
	return rd.ReadNAtFast(min, max-min)
}
func ReadFullCopy(rd Reader) ([]byte, error) {
	min, max := rd.Min(), rd.Max()
	return rd.ReadNAtCopy(min, max-min)
}

//----------

func SetBytes(rw ReadWriter, b []byte) error {
	min, max := rw.Min(), rw.Max()
	return rw.Overwrite(min, max-min, b)
}
func SetString(rw ReadWriter, s string) error {
	return SetBytes(rw, []byte(s))
}

//----------

// Iterate over n+1 runes, with the last rune being eofRune(-1).
func ReaderIter(r Reader, fn func(i int, ru rune) bool) error {
	for i := r.Min(); ; {
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

//----------

func HasPrefix(r Reader, i int, s []byte) bool {
	if len(s) == 0 {
		return true
	}
	b, err := r.ReadNAtFast(i, len(s))
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
	// TODO: ignore accents?

	if chunk < len(sep) {
		return -1, fmt.Errorf("chunk smaller then sep")
	}

	// ignore case
	if toLower {
		sep = ToLowerAsciiCopy(sep) // copy
	}

	b := r.Max()
	for k := i; k < b; k += chunk - (len(sep) - 1) {
		c := chunk
		if c > b-k {
			c = b - k
		}

		j, err := indexCtx3(r, k, c, sep, toLower)
		if err != nil || j >= 0 {
			return j, err
		}

		// check context cancelation
		if err := ctx.Err(); err != nil {
			return 0, err
		}
	}

	return -1, nil
}

func indexCtx3(r Reader, i, n int, sep []byte, toLower bool) (int, error) {
	p, err := r.ReadNAtFast(i, n)
	if err != nil {
		return 0, err
	}

	// ignore case
	if toLower {
		p = ToLowerAsciiCopy(p) // copy
	}

	j := bytes.Index(p, sep)
	if j >= 0 {
		return i + j, nil
	}
	return -1, nil
}

// Lower case at byte level without expanding in size the resulting byte slice.
func ToLowerAsciiCopy(p []byte) []byte {
	// bytes.ToLower expands the size of the returning slice.
	//return bytes.ToLower(p) // copy

	u := make([]byte, len(p))
	for i := 0; i < len(p); i++ {
		c := p[i]
		if 'A' <= c && c <= 'Z' {
			u[i] = c + ('a' - 'A')
		} else {
			u[i] = c
		}
	}
	return u
}

//----------

// On error, returns best failing index. Use errors.Is(err, io.EOF) to handle limitedreaders.
func IndexFunc(r Reader, i int, truth bool, f func(rune) bool) (index, size int, err error) {
	for {
		ru, size, err := r.ReadRuneAt(i)
		if err != nil {
			// improve invalid index
			m := r.Max()
			if i > m {
				i = m
			}

			return i, 0, err
		}
		if f(ru) == truth {
			return i, size, nil
		}
		i += size
	}
}

// On error, returns best failing index. Use errors.Is(err, io.EOF) to handle limitedreaders.
func LastIndexFunc(r Reader, i int, truth bool, f func(rune) bool) (index, size int, err error) {
	for {
		ru, size, err := r.ReadLastRuneAt(i)
		if err != nil {
			// improve invalid index
			m := r.Min()
			if i < m {
				i = m
			}

			return i, 0, err
		}
		i -= size
		if f(ru) == truth {
			return i, size, nil
		}
	}
}

//----------

// Returns index where truth was found.
func ExpandIndexFunc(r Reader, i int, truth bool, f func(rune) bool) int {
	j, _, _ := IndexFunc(r, i, truth, f)
	return j // found, or last known index before an err
}

// Returns last index before truth was found.
func ExpandLastIndexFunc(r Reader, i int, truth bool, f func(rune) bool) int {
	j, size, err := LastIndexFunc(r, i, truth, f)
	if err != nil {
		return j // last known index before an err
	}
	return j + size
}

//----------

func LineStartIndex(r Reader, i int) (int, error) {
	k, size, err := NewLineLastIndex(r, i)
	if errors.Is(err, io.EOF) {
		return k, nil
	}
	return k + size, err
}

// index after '\n' (with isNewLine true), or max index
func LineEndIndex(r Reader, i int) (int, bool, error) {
	k, size, err := NewLineIndex(r, i)
	if errors.Is(err, io.EOF) {
		return k, false, nil
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

// Also used at: selectword, movecursorjump{left,right}
func IsWordRune(ru rune) bool {
	return unicode.IsLetter(ru) || unicode.IsDigit(ru) || ru == '_'
}

func WordAtIndex(r Reader, index int) ([]byte, int, error) {
	// right side
	i1, _, err := IndexFunc(r, index, false, IsWordRune)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, 0, err
	}
	if i1 == index { // don't match word at index
		return nil, 0, errors.New("word not found")
	}

	// left side
	i0, size, err := LastIndexFunc(r, index, false, IsWordRune)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, 0, err
	}
	i0 += size

	w, err := r.ReadNAtCopy(i0, i1-i0)
	if err != nil {
		return nil, 0, err
	}
	return w, i0, nil
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
