package tautil

func DuplicateLines(ta Texta) {
	a, b, ok := linesStringIndexes(ta)
	if !ok {
		return
	}
	t := ta.Str()[a:b]
	ta.EditInsert(b, t)
	ta.EditCommit()
	ta.SetSelectionOn(true)
	ta.SetSelectionIndex(b)

	_, b2, ok := PreviousRuneIndex(t, len(t))
	if !ok {
		panic("!")
	}
	b += b2
	ta.SetCursorIndex(b)
}
