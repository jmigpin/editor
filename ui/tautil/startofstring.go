package tautil

func StartOfString(ta Texta, sel bool) {
	activateSelection(ta, sel)
	defer deactivateSelectionCheck(ta)
	ta.SetCursorIndex(0)
	ta.MakeIndexVisible(0)
}
