package tautil

import "strings"

func Comment(ta Texta) {
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
		ta.SetSelectionOn(false)
		// move cursor to the right due to inserted runes
		i := strings.Index(ta.Str()[a:a+len(str)], "//")
		if i >= 0 {
			ci := ta.CursorIndex()
			if ci >= a+i {
				ta.SetCursorIndex(ci + 2)
			}
		}
	} else {
		ta.SetSelectionOn(true)
		ta.SetSelectionIndex(a)
		ta.SetCursorIndex(a + len(str))
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
		ta.SetSelectionOn(false)
		// move cursor to the left due to deleted runes
		i := strings.IndexFunc(ta.Str()[a:a+len(str)], isNotSpace)
		if i >= 0 {
			ci := ta.CursorIndex()
			if ci > a+i {
				ta.SetCursorIndex(ci - 2)
			}
		}
	} else {
		ta.SetSelectionOn(true)
		ta.SetSelectionIndex(a)
		ta.SetCursorIndex(a + len(str))
	}
}
