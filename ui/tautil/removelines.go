package tautil

func RemoveLines(ta Texta) {
	a, b, _ := linesStringIndexes(ta)
	ta.EditOpen()
	ta.EditDelete(a, b)
	ta.EditClose()
	ta.SetSelectionOn(false)
	ta.SetCursorIndex(a)
}
