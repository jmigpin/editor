package tautil

func TabRight(ta Texta) {
	_ = alterSelectedText(ta, tabRightLines)
}
func tabRightLines(str string) (string, bool) {
	// assume it's at a line start
	for i := 0; i < len(str); {
		str = str[:i] + string('\t') + str[i:] // insert at start of line
		i, _ = lineEndIndexNextIndex(str, i)
	}
	return str, true
}

func TabLeft(ta Texta) {
	a, b, _ := linesStringIndexes(ta)

	str := ta.Str()[a:b]

	// remove from line start
	nlines := 0
	altered := false
	for i := 0; i < len(str); {
		nlines++
		if str[i] == '\t' || str[i] == ' ' {
			altered = true
			str = str[:i] + str[i+1:] // +1 is length of '\t' or ' '
		}
		i, _ = lineEndIndexNextIndex(str, i)
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
			ci--
		}
		ta.SetCursorIndex(ci)
	} else {
		ta.SetSelectionOn(true)
		ta.SetSelectionIndex(a)
		ta.SetCursorIndex(a + len(str))
	}
}
