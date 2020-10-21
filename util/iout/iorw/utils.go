package iorw

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"unicode"
)

//godebug:annotatefile

func MakeBytesCopy(b []byte) []byte {
	p := make([]byte, len(b), len(b))
	copy(p, b)
	return p
}

//----------

func NewStringReaderAt(s string) ReaderAt {
	return NewBytesReadWriterAt([]byte(s))
}

//----------

func REqual(r ReaderAt, i, n int, p []byte) (bool, error) {
	if n != len(p) {
		return false, nil
	}
	b, err := r.ReadFastAt(i, n)
	if err != nil {
		return false, err
	}
	return bytes.Equal(b, p), nil
}

//----------

// Result might not be a copy.
func ReadFastFull(rd ReaderAt) ([]byte, error) {
	min, max := rd.Min(), rd.Max()
	return rd.ReadFastAt(min, max-min)
}

// Result might not be a copy.
func ReadFullCopy(rd ReaderAt) ([]byte, error) {
	b, err := ReadFastFull(rd)
	if err != nil {
		return nil, err
	}
	return MakeBytesCopy(b), nil
}

//----------

func SetBytes(rw ReadWriterAt, b []byte) error {
	return rw.OverwriteAt(rw.Min(), rw.Max(), b)
}
func SetString(rw ReadWriterAt, s string) error {
	return SetBytes(rw, []byte(s))
}
func Append(rw ReadWriterAt, b []byte) error {
	return rw.OverwriteAt(rw.Max(), 0, b)
}

//----------

const EndRune = -1

// Iterate over n+1 runes, with the last rune being eofRune(-1).
func ReaderIter(r ReaderAt, fn func(i int, ru rune) bool) error {
	for i := r.Min(); ; {
		ru, size, err := ReadRuneAt(r, i)
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

//----------

func HasPrefix(r ReaderAt, i int, s []byte) bool {
	if len(s) == 0 {
		return true
	}
	b, err := r.ReadFastAt(i, len(s))
	if err != nil {
		return false
	}
	return bytes.HasPrefix(b, s)
}

//----------

func Index(r ReaderAt, i int, sep []byte, toLower bool) (int, error) {
	ctx := context.Background()
	return IndexCtx(ctx, r, i, sep, toLower)
}

// Returns (-1, nil) if not found.
func IndexCtx(ctx context.Context, r ReaderAt, i int, sep []byte, toLower bool) (int, error) {
	return indexCtx2(ctx, r, i, sep, toLower, 32*1024)
}

func indexCtx2(ctx context.Context, r ReaderAt, i int, sep []byte, toLower bool, chunk int) (int, error) {
	// TODO: ignore accents? use strings (runes)

	if chunk < len(sep) {
		return -1, fmt.Errorf("chunk smaller then sep")
	}

	// ignore case
	if toLower {
		sep = ToLowerAsciiCopy(sep) // copy
	}

	m := r.Max()
	for k := i; k < m; k += chunk - (len(sep) - 1) {
		c := chunk
		if c > m-k {
			c = m - k
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

func indexCtx3(r ReaderAt, i, n int, sep []byte, toLower bool) (int, error) {
	p, err := r.ReadFastAt(i, n)
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
func RuneIndexFn(r ReaderAt, i int, truth bool, f func(rune) bool) (index, size int, err error) {
	for {
		ru, size, err := ReadRuneAt(r, i)
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
func RuneLastIndexFn(r ReaderAt, i int, truth bool, f func(rune) bool) (index, size int, err error) {
	for {
		ru, size, err := ReadLastRuneAt(r, i)
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

//// Returns index where truth was found.
func ExpandRuneIndexFn(r ReaderAt, i int, truth bool, f func(rune) bool) int {
	j, _, _ := RuneIndexFn(r, i, truth, f)
	return j // found, or last known index before an err
}

// Returns last index before truth was found.
func ExpandRuneLastIndexFn(r ReaderAt, i int, truth bool, f func(rune) bool) int {
	j, size, err := RuneLastIndexFn(r, i, truth, f)
	if err != nil {
		return j // last known index before an err
	}
	return j + size
}

//----------

func LinesIndexes(r ReaderAt, a, b int) (int, int, bool, error) {
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

func LineStartIndex(r ReaderAt, i int) (int, error) {
	k, size, err := NewlineLastIndex(r, i)
	if errors.Is(err, io.EOF) {
		return k, nil
	}
	return k + size, err
}

// index after '\n' (with isNewLine true), or max index
func LineEndIndex(r ReaderAt, i int) (int, bool, error) {
	k, size, err := NewlineIndex(r, i)
	if errors.Is(err, io.EOF) {
		return k, false, nil
	}
	isNewLine := err == nil
	return k + size, isNewLine, err
}

//----------

func isNewline(ru rune) bool { return ru == '\n' }

func NewlineIndex(r ReaderAt, i int) (int, int, error) {
	return RuneIndexFn(r, i, true, isNewline)
}

func NewlineLastIndex(r ReaderAt, i int) (int, int, error) {
	return RuneLastIndexFn(r, i, true, isNewline)
}

//----------

// Also used at: selectword, movecursorjump{left,right}
func IsWordRune(ru rune) bool {
	return unicode.IsLetter(ru) || unicode.IsDigit(ru) || ru == '_'
}

func WordAtIndex(r ReaderAt, index int) ([]byte, int, error) {
	// right side
	i1, _, err := RuneIndexFn(r, index, false, IsWordRune)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, 0, err
	}
	if i1 == index { // don't match word at index
		return nil, 0, errors.New("word not found")
	}

	// left side
	i0, size, err := RuneLastIndexFn(r, index, false, IsWordRune)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, 0, err
	}
	i0 += size

	w, err := r.ReadFastAt(i0, i1-i0)
	if err != nil {
		return nil, 0, err
	}
	return MakeBytesCopy(w), i0, nil
}

func WordIsolated(r ReaderAt, i, le int) bool {
	// previous rune can't be a word rune
	ru, _, err := ReadLastRuneAt(r, i)
	if err == nil && IsWordRune(ru) {
		return false
	}
	// next rune can't be a word rune
	ru, _, err = ReadRuneAt(r, i+le)
	if err == nil && IsWordRune(ru) {
		return false
	}
	return true
}
