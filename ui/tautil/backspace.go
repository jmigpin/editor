package tautil

func Backspace(ta Texta) {
	var a, b int
	var ok bool
	if ta.SelectionOn() {
		a, b = SelectionStringIndexes(ta)
		ta.SetSelectionOff()
	} else {
		b = ta.CursorIndex()
		_, a, ok = PreviousRuneIndex(ta.Str(), b)
		if !ok {
			return
		}
	}
	ta.EditOpen()
	defer ta.EditCloseAfterSetCursor()
	ta.EditDelete(a, b)
	ta.SetCursorIndex(a)
}
