package tautil

func Cut(ta Texta) {
	if !ta.SelectionOn() {
		return
	}
	a, b := SelectionStringIndexes(ta)
	ta.SetClipboardString(ta.Str()[a:b])
	ta.EditRemove(a, b)
	ta.EditDone()
	ta.SetSelectionOn(false)
	ta.SetCursorIndex(a)
}
