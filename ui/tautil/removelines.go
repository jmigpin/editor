package tautil

func RemoveLines(ta Texta) {
	a, b, _ := linesStringIndexes(ta)
	ta.EditOpen()
	defer ta.EditCloseAfterSetCursor()
	ta.EditDelete(a, b)
	ta.SetSelectionOff()
	ta.SetCursorIndex(a)
}
