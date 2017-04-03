package tautil

func DuplicateLines(ta Texta) {
	a, b, hasNewline := linesStringIndexes(ta)
	s := ta.Str()[a:b]
	ta.EditOpen()
	if !hasNewline {
		ta.EditInsert(b, "\n")
		b++
	}
	ta.EditInsert(b, s)
	ta.EditClose()

	// cursor index without the newline
	c := previousRuneIndexIfLastIsNewline(s)

	ta.SetSelection(b, b+c)
}
