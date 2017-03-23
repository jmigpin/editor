package tautil

func TabRight(ta Texta) {
	if !ta.SelectionOn() {
		InsertRune(ta, '\t')
		return
	}

	a, b, _ := linesStringIndexes(ta)

	str := ta.Str()[a:b]

	// insert at line start
	for i := 0; i < len(str); i, _ = lineEndIndexNextIndex(str, i) {
		str = str[:i] + string('\t') + str[i:]
	}

	// replace
	ta.EditOpen()
	ta.EditDelete(a, b)
	ta.EditInsert(a, str)
	ta.EditClose()

	ta.SetSelectionOn(true)
	ta.SetSelectionIndex(a)
	ta.SetCursorIndex(a + len(str))
}
func TabLeft(ta Texta) {
	a, b, _ := linesStringIndexes(ta)

	str := ta.Str()[a:b]

	// remove from line start
	nlines := 0
	altered := false
	for i := 0; i < len(str); i, _ = lineEndIndexNextIndex(str, i) {
		if str[i] == '\t' || str[i] == ' ' {
			altered = true
			str = str[:i] + str[i+1:] // +1 is length of '\t' or ' '
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
		ci := ta.CursorIndex()
		if ci > a {
			// move cursor to the left due to removed rune
			ta.SetCursorIndex(ci - 1)
		}
	} else {
		ta.SetSelectionOn(true)
		ta.SetSelectionIndex(a)
		ta.SetCursorIndex(a + len(str))
	}
}
