package parseutil

import (
	"fmt"
	"runtime"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/util/statemach"
)

var FilenameEscapeRunes string

func init() {
	if runtime.GOOS == "windows" {
		FilenameEscapeRunes = " %?<>()^"
	} else {
		FilenameEscapeRunes = " :%?<>()\\"
	}
}

//----------

func isFilenameRune(ru rune) bool {
	return unicode.IsLetter(ru) || unicode.IsDigit(ru) ||
		strings.ContainsRune(`_/~\-\.\\^ `, ru)
}

//----------

func EscapeFilename(str string) string {
	w := []rune{}
	for _, ru := range str {
		if strings.ContainsRune(FilenameEscapeRunes, ru) {
			w = append(w, EscapeRune)
		}
		w = append(w, ru)
	}
	return string(w)
}

//----------

func AcceptAdvanceFilename(s *statemach.String) (string, bool) {
	r := s.AcceptLoopFn(func(ru rune) bool {
		if s.IsEscapeAccept(ru, EscapeRunes) {
			return true
		}
		return isFilenameRune(ru)
	})
	if !r {
		return "", false
	}
	filename := s.Value()
	s.Advance()
	return filename, true
}

//----------

func ExpandLastIndexOfFilenameFmt(str string, max int) int {
	esc := false
	w := []rune{}
	isOk := func(ru rune) bool {
		if !esc && strings.ContainsRune(FilenameEscapeRunes, ru) {
			esc = true
			w = append(w, ru)
			return true
		}
		if esc {
			// allow expanding ':' without escaping
			isColon := w[len(w)-1] == ':'

			if ru == EscapeRune || isColon {
				esc = false
				w = []rune{}
				return true
			}

			return false
		}
		return isFilenameRune(ru)
	}

	i := ExpandLastIndexFunc(str, max, false, isOk)
	if i < 0 {
		return -1
	}
	if len(w) > 0 {
		i += len(string(w))
	}
	return i
}

//----------

// TODO: unify into one struct

type FilePos struct {
	Filename     string
	Line, Column int // bigger than zero to be considered
}
type FileOffset struct {
	Filename string
	Offset   int
	Len      int // length after offset for a range
}

//----------

// Parse fmt: <filename:line?:col?>. Accepts escapes but doesn't unescape.
func ParseFilePos(str string) (*FilePos, error) {
	s := statemach.NewString(str)

	// filename
	filename, ok := AcceptAdvanceFilename(s)
	if !ok {
		return nil, fmt.Errorf("expecting filename")
	}
	fp := &FilePos{Filename: filename}

	// ":"
	if !s.AcceptAny(":") {
		return fp, nil
	}
	s.Advance()

	// line
	if !s.AcceptInt() {
		return fp, nil
	}
	line, err := s.ValueInt()
	if err != nil {
		return fp, nil // not returning err
	}
	s.Advance()
	fp.Line = line

	// ":"
	if !s.AcceptAny(":") {
		return fp, nil
	}
	s.Advance()

	// column
	if !s.AcceptInt() {
		return fp, nil
	}
	col, err := s.ValueInt()
	if err != nil {
		return fp, nil // not returning err
	}
	s.Advance()
	fp.Column = col

	return fp, nil
}
