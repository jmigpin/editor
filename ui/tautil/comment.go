package tautil

import (
	"strings"
	"unicode"
)

func Comment(ta Texta) {
	a, b, _ := linesStringIndexes(ta)

	str := ta.Str()[a:b]

	IsNotASpaceExceptNewLine := func(ru rune) bool {
		return !unicode.IsSpace(ru) || ru == '\n'
	}

	// find comment insertion index
	start := 1000
	for i := 0; i < len(str); i, _ = lineEndIndexNextIndex(str, i) {
		j := strings.IndexFunc(str[i:], IsNotASpaceExceptNewLine)
		if j < 0 {
			break
		}
		// ignore empty lines
		w, _ := lineEndIndexNextIndex(str, i)
		if strings.TrimSpace(str[i:w]) == "" {
			continue
		}
		if j < start {
			start = j
		}
	}

	altered := false
	nlines := 0
	for i := 0; i < len(str); i, _ = lineEndIndexNextIndex(str, i) {
		// ignore empty lines
		w, _ := lineEndIndexNextIndex(str, i)
		if strings.TrimSpace(str[i:w]) == "" {
			continue
		}

		i += start

		// insert comment
		str = str[:i] + "//" + str[i:]
		altered = true

		nlines++
	}

	if !altered {
		return
	}

	// replace
	ta.EditOpen()
	ta.EditDelete(a, b)
	ta.EditInsert(a, str)
	ta.EditClose()

	if nlines <= 1 {
		ta.SetSelectionOff()
		// move cursor to the right due to inserted runes
		i := strings.Index(ta.Str()[a:a+len(str)], "//")
		if i >= 0 {
			ci := ta.CursorIndex()
			if ci >= a+i {
				ta.SetCursorIndex(ci + 2)
			}
		}
	} else {
		ta.SetSelection(a, a+len(str))
	}
}
func Uncomment(ta Texta) {
	a, b, _ := linesStringIndexes(ta)

	str := ta.Str()[a:b]
	altered := false
	nlines := 0
	for i := 0; i < len(str); i, _ = lineEndIndexNextIndex(str, i) {
		// first non space rune: possible multiline jump
		j := strings.IndexFunc(str[i:], isNotSpace)
		if j < 0 {
			break
		}
		i += j

		// remove comment
		if strings.HasPrefix(str[i:], "//") {
			altered = true
			str = str[:i] + str[i+len("//"):]
		}

		nlines++
	}

	if !altered {
		return
	}

	// replace
	ta.EditOpen()
	ta.EditDelete(a, b)
	ta.EditInsert(a, str)
	ta.EditClose()

	if nlines <= 1 {
		ta.SetSelectionOff()
		// move cursor to the left due to deleted runes
		i := strings.IndexFunc(ta.Str()[a:a+len(str)], isNotSpace)
		if i >= 0 {
			ci := ta.CursorIndex()
			if ci > a+i {
				ta.SetCursorIndex(ci - 2)
			}
		}
	} else {
		ta.SetSelection(a, a+len(str))
	}
}
