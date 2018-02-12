package tautil

func TabRight(ta Texta) {
	if !ta.SelectionOn() {
		InsertString(ta, "\t")
		return
	}

	a, b, _ := linesStringIndexes(ta)

	str := ta.Str()[a:b]

	// insert at line start
	for i := 0; i < len(str); i, _ = LineEndIndexNextIndex(str, i) {
		str = str[:i] + string('\t') + str[i:]
	}

	// replace
	ta.EditOpen()
	defer ta.EditCloseAfterSetCursor()
	ta.EditDelete(a, b)
	ta.EditInsert(a, str)

	// don't select newline as last char
	c := previousRuneIndexIfLastIsNewline(str)

	ta.SetSelection(a, a+c)
}
func TabLeft(ta Texta) {
	a, b, _ := linesStringIndexes(ta)

	str := ta.Str()[a:b]

	// remove from line start
	altered := false
	deletions := 0
	for i := 0; i < len(str); i, _ = LineEndIndexNextIndex(str, i) {
		if str[i] == '\t' || str[i] == ' ' {
			altered = true
			deletions++
			str = str[:i] + str[i+1:] // +1 is length of '\t' or ' '
		}
	}

	if !altered {
		return
	}

	// replace
	ta.EditOpen()
	defer ta.EditCloseAfterSetCursor()
	ta.EditDelete(a, b)
	ta.EditInsert(a, str)

	// don't select newline as last char
	c := previousRuneIndexIfLastIsNewline(str)

	ta.SetSelection(a, a+c)
}
