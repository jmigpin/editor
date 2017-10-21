package tautil

func Cut(ta Texta) {
	if !ta.SelectionOn() {
		return
	}
	a, b := SelectionStringIndexes(ta)
	ta.SetClipboardCopy(ta.Str()[a:b])
	ta.EditOpen()
	defer ta.EditCloseAfterSetCursor()
	ta.EditDelete(a, b)
	ta.SetSelectionOff()
	ta.SetCursorIndex(a)
}
