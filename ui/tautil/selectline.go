package tautil

func SelectLine(ta Texta) {
	ta.SetSelectionOn(false)
	a, b, _ := linesStringIndexes(ta)
	ta.SetSelectionOn(true)
	ta.SetSelectionIndex(a)
	ta.SetCursorIndex(b)
}
