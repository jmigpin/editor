package tautil

func Backspace(ta Texta) {
	var a, b int
	var ok bool
	if ta.SelectionOn() {
		a, b, ok = selectionStringIndexes(ta)
		if !ok {
			return
		}
		ta.SetSelectionOn(false)
	} else {
		b = ta.CursorIndex()
		_, a, ok = PreviousRuneIndex(ta.Str(), b)
		if !ok {
			return
		}
	}
	ta.EditRemove(a, b)
	ta.EditCommit()
	ta.SetCursorIndex(a)
}
