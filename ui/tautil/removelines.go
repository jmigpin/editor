package tautil

func RemoveLines(ta Texta) {
	a, b, ok := linesStringIndexes(ta)
	if !ok {
		return
	}
	ta.EditRemove(a, b)
	ta.EditCommit()
	ta.SetSelectionOn(false)
	ta.SetCursorIndex(a)
}
