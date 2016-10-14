package tautil

func RemoveLines(ta Texta) {
	a, b, ok := linesStringIndexes(ta)
	if !ok {
		return
	}
	ta.EditRemove(a, b)
	ta.EditDone()
	ta.SetSelectionOn(false)
	ta.SetCursorIndex(a)
}
