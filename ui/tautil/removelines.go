package tautil

func RemoveLines(ta Texta) {
	a, b, _ := linesStringIndexes(ta)
	ta.EditRemove(a, b)
	ta.EditDone()
	ta.SetSelectionOn(false)
	ta.SetCursorIndex(a)
}
