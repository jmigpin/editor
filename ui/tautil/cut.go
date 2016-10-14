package tautil

func Cut(ta Texta) {
	if !ta.SelectionOn() {
		return
	}
	a, b, ok := selectionStringIndexes(ta)
	if !ok {
		return
	}
	ta.SetClipboardString(ta.Str()[a:b])
	ta.EditRemove(a, b)
	ta.EditCommit()
	ta.SetSelectionOn(false)
	ta.SetCursorIndex(a)
}
