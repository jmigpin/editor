package parseutil

import (
	"errors"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/osutil"
)

const QuoteRunes = "\"'`"

//----------

func IndexFunc(s string, truth bool, f func(rune) bool) (index, size int) {
	l := len(s)
	for i := 0; i < l; {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError {
			break
		}
		if f(r) == truth {
			return i, size
		}
		i += size
	}
	return -1, 0
}

func LastIndexFunc(s string, truth bool, f func(rune) bool) (index, size int) {
	for i := len(s); i > 0; {
		r, size := utf8.DecodeLastRuneInString(s[:i])
		if r == utf8.RuneError {
			break
		}
		i -= size
		if f(r) == truth {
			return i, size
		}
	}
	return -1, 0
}

//----------

// Returns -1 if max was passed.
func ExpandIndexFunc(str string, max int, truth bool, f func(rune) bool) int {
	c := 0
	f2 := func(ru rune) bool {
		c++
		if c > max {
			return truth
		}
		return f(ru)
	}
	i, _ := IndexFunc(str, truth, f2)
	if c > max {
		return -1
	}
	if i < 0 {
		i = len(str)
	}
	return i
}

// Returns -1 if max was passed.
func ExpandLastIndexFunc(str string, max int, truth bool, f func(rune) bool) int {
	c := 0
	f2 := func(ru rune) bool {
		c++
		if c > max {
			return truth
		}
		return f(ru)
	}
	i, size := LastIndexFunc(str, truth, f2)
	if c > max {
		return -1
	}
	if i < 0 {
		i = 0
	} else {
		i += size // next rune
	}
	return i
}

//----------

func LineStartIndex(str string, index int) int {
	i := strings.LastIndex(str[:index], "\n")
	if i < 0 {
		i = 0
	} else {
		i += 1 // rune length of '\n'
	}
	return i
}

func LineEndIndexNextIndex(str string, index int) (_ int, hasNewline bool) {
	i := strings.Index(str[index:], "\n")
	if i < 0 {
		return len(str), false
	}
	return index + i + 1, true // 1 is "\n" size
}

//----------

func LineColumnIndex(str string, line, column int) int {
	line--
	column--

	// rune index of line/column
	index := -1
	l, c := 0, 0
	for ri, ru := range str {
		if l == line {
			if c == column {
				index = ri
				break
			}
			c++
		}
		if ru == '\n' {
			l++
			if l == line {
				index = ri + 1 // column 0 (+1 is to pass '\n')
			} else if l > line {
				break
			}
		}
	}
	return index
}

func IndexLineColumn(str string) (int, int) {
	line, lineStart := 0, 0
	for ri, ru := range str {
		if ru == '\n' {
			line++
			lineStart = ri
		}
	}
	col := len(str) - lineStart
	line++
	return line, col
}

//----------

func UnescapeString(str string) string {
	w := []rune{}
	esc := false
	for _, ru := range str {
		if !esc && strings.ContainsRune(osutil.EscapeRunes, ru) {
			esc = true
			continue
		}
		if esc {
			esc = false
		}
		w = append(w, ru)
	}
	return string(w)
}
func UnescapeRunes(str, escapable string) string {
	w := []rune{}
	esc := false
	for _, ru := range str {
		if !esc && strings.ContainsRune(osutil.EscapeRunes, ru) {
			esc = true
			continue
		}
		if esc {
			esc = false

			// re-add escape rune if not one of the escapable runes
			if !strings.ContainsRune(escapable, ru) {
				w = append(w, osutil.EscapeRune)
			}
		}
		w = append(w, ru)
	}
	return string(w)
}

//----------

func IsWordRune(ru rune) bool {
	return unicode.IsLetter(ru) || unicode.IsDigit(ru) || ru == '_'
}

//----------

func WordAtIndex(r iout.Reader, index, max int) ([]byte, int, error) {
	ErrWordNotFound := errors.New("word not found")

	// right side
	i1, _, err := iout.IndexFunc(r, index, max, false, IsWordRune)
	if err != nil {
		if err == io.EOF {
			i1 = r.Len()
		} else {
			return nil, 0, err
		}
	}
	if i1 == index { // don't match word at index
		return nil, 0, ErrWordNotFound
	}

	// left side
	i0, size, err := iout.LastIndexFunc(r, index, max, false, IsWordRune)
	if err != nil {
		if err == io.EOF {
			i0 = 0
		} else {
			return nil, 0, err
		}
	} else {
		i0 += size
	}

	s, err := r.ReadNAt(i0, i1-i0)
	if err != nil {
		return nil, 0, err
	}

	return s, i0, nil
}

func WordIsolated(r iout.Reader, i, le int) bool {
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

//----------

func DetectEnvVar(str, name string) bool {
	vstr := "$" + name
	i := strings.Index(str, vstr)
	if i < 0 {
		return false
	}

	e := i + len(vstr)
	if e > len(str) {
		return false
	}

	// validate rune after the name
	ru, _ := utf8.DecodeRuneInString(str[e:])
	if ru != utf8.RuneError {
		if unicode.IsLetter(ru) || unicode.IsDigit(ru) || ru == '_' {
			return false
		}
	}

	return true
}
