package tautil

func EndOfString(ta Texta, sel bool) {
	activateSelection(ta, sel)
	i := len(ta.Text())
	ta.SetCursorIndex(i)
	ta.MakeIndexVisible(i)
	deactivateSelectionCheck(ta)
}
