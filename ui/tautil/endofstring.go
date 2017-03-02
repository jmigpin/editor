package tautil

func EndOfString(ta Texta, sel bool) {
	activateSelection(ta, sel)
	defer deactivateSelectionCheck(ta)
	i := len(ta.Str())
	ta.SetCursorIndex(i)
	ta.MakeIndexVisible(i)
}
