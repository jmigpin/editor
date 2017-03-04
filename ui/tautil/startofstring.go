package tautil

func StartOfString(ta Texta, sel bool) {
	updateSelectionState(ta, sel)
	ta.SetCursorIndex(0)
	ta.MakeIndexVisible(0)
}
