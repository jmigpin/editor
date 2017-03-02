package tautil

func DuplicateLines(ta Texta) {
	a, b, hasNewline := linesStringIndexes(ta)
	t := ta.Str()[a:b]
	if !hasNewline {
		ta.EditInsert(b, "\n")
		b++
	}
	ta.EditInsert(b, t)
	ta.EditDone()
	ta.SetSelectionOn(true)
	ta.SetSelectionIndex(b)

	_, b2, ok := PreviousRuneIndex(t, len(t))
	if !ok {
		panic("!")
	}
	b += b2
	ta.SetCursorIndex(b)
}
