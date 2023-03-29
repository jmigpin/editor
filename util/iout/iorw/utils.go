package iorw

import (
	"bytes"
	"errors"
	"io"
	"unicode"

	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/parseutil/pscan"
)

func NewScanner(rd ReaderAt) *pscan.Scanner {
	sc := pscan.NewScanner()
	src, err := ReadFastFull(rd)
	if err != nil {
		//return nil, err // TODO
		return sc // best effort, returns empty scanner
	}
	sc.SetSrc2(src, rd.Min())
	return sc
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
	return iout.CopyBytes(b), nil
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

func HasSuffix(r ReaderAt, i int, s []byte) bool {
	//godebug:annotateblock
	if len(s) == 0 {
		return true
	}
	b, err := r.ReadFastAt(i-len(s), len(s))
	if err != nil {
		return false
	}
	return bytes.HasSuffix(b, s)
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

// Returns index where truth was found.
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

//func Lines(r ReaderAt, a, b int) (int, int, [][]byte, error) {
//	ls, err := LineStartIndex(r, a)
//	if err != nil {
//		return 0, 0, false, err
//	}

//	le, newline, err := LineEndIndex(r, b)
//	if err != nil {
//		return 0, 0, false, err
//	}
//	lines := [][]byte{}
//	for i := ls; i <= b; {
//		le, newline, err := LineEndIndex(r, i)
//		if err != nil {
//			break
//		}
//		line:=
//		lines=append(lines, line)
//	}
//	return ls, le, newline, nil
//}

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
	return iout.CopyBytes(w), i0, nil
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
