package tautil

func EndOfString(ta Texta, sel bool) {
	updateSelectionState(ta, sel)
	i := len(ta.Str())
	ta.SetCursorIndex(i)
}
