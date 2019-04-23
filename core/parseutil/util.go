package parseutil

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/statemach"
)

//----------

func AddEscapes(str string, escape rune, escapeRunes string) string {
	w := []rune{}
	for _, ru := range str {
		if strings.ContainsRune(escapeRunes, ru) {
			w = append(w, escape)
		}
		w = append(w, ru)
	}
	return string(w)
}

func RemoveEscapes(str string, escape rune) string {
	w := []rune{}
	esc := false
	for _, ru := range str {
		if !esc {
			if ru == escape {
				esc = true
				continue
			}
		} else {
			esc = false
		}
		w = append(w, ru)
	}
	return string(w)
}

// removes the escape only if escapable
func RemoveEscapesEscapable(str string, escape rune, escapable string) string {
	w := []rune{}
	esc := false
	for _, ru := range str {
		if !esc {
			if ru == escape {
				esc = true
				continue
			}
		} else {
			esc = false

			// re-add escape if not one of the escapable
			if !strings.ContainsRune(escapable, ru) {
				w = append(w, escape)
			}
		}
		w = append(w, ru)
	}
	return string(w)
}

//----------

func ExpandIndexesEscape(rd iorw.Reader, index int, truth bool, fn func(rune) bool, escape rune) (int, int) {
	// ensure the index is not in the middle of an escape
	index = ImproveExpandIndexEscape(rd, index, escape)

	l := ExpandLastIndexEscape(rd, index, false, fn, escape)
	r := ExpandIndexEscape(rd, index, false, fn, escape)
	return l, r
}

func ExpandIndexEscape(r iorw.Reader, i int, truth bool, fn func(rune) bool, escape rune) int {
	sc := statemach.NewScanner(r)
	sc.Pos = i
	return expandEscape(sc, truth, fn, escape)
}

func ExpandLastIndexEscape(r iorw.Reader, i int, truth bool, fn func(rune) bool, escape rune) int {
	sc := statemach.NewScanner(r)
	sc.Pos = i

	// read direction
	tmp := sc.Reverse
	sc.Reverse = true
	defer func() { sc.Reverse = tmp }() // restore

	return expandEscape(sc, truth, fn, escape)
}

func expandEscape(sc *statemach.Scanner, truth bool, fn func(rune) bool, escape rune) int {
	for {
		if sc.Match.End() {
			break
		}
		if sc.Match.Escape(escape) {
			continue
		}
		u := sc.Pos
		ru := sc.ReadRune()
		if fn(ru) == truth {
			sc.Pos = u
			break
		}
	}
	return sc.Pos
}

//----------

func ImproveExpandIndexEscape(r iorw.Reader, i int, escape rune) int {
	sc := statemach.NewScanner(r)
	sc.Pos = i

	// read direction
	tmp := sc.Reverse
	sc.Reverse = true
	defer func() { sc.Reverse = tmp }() // restore

	for {
		if sc.Match.End() {
			break
		}
		if sc.Match.Rune(escape) {
			continue
		}
		break
	}
	return sc.Pos
}

//----------

func LineColumnIndex(rd iorw.Reader, line, column int) (int, error) {
	// must have a good line
	if line <= 0 {
		return 0, fmt.Errorf("bad line: %v", line)
	}
	line-- // make line 0 the first line

	// tolerate bad columns
	if column <= 0 {
		column = 1
	}
	column-- // make column 0 the first column

	index := -1
	l, lStart := 0, 0
	ri := 0
	for {
		if l == line {
			index = ri
			c := ri - lStart
			if c >= column {
				break
			}
		} else if l > line {
			break
		}

		ru, size, err := rd.ReadRuneAt(ri)
		if err != nil {
			return 0, err
		}
		ri += size
		if ru == '\n' {
			l++
			lStart = ri
		}
	}
	if index < 0 {
		return 0, fmt.Errorf("line not found: %v", line)
	}
	return index, nil
}

func IndexLineColumn(rd iorw.Reader, index int) (int, int, error) {
	line, lineStart := 0, 0
	ri := 0
	for ri < index {
		ru, size, err := rd.ReadRuneAt(ri)
		if err != nil {
			return 0, 0, err
		}
		ri += size
		if ru == '\n' {
			line++
			lineStart = ri
		}
	}
	line++                    // first line is 1
	col := ri - lineStart + 1 // first column is 1
	return line, col, nil
}

//----------

// TODO: review
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
