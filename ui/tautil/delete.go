package tautil

func Delete(ta Texta) {
	var a, b int
	if ta.SelectionOn() {
		a, b = SelectionStringIndexes(ta)
		ta.SetSelectionOn(false)
	} else {
		var ok bool
		a = ta.CursorIndex()
		_, b, ok = NextRuneIndex(ta.Str(), a)
		if !ok {
			return
		}
	}
	ta.EditRemove(a, b)
	ta.EditDone()
	ta.SetCursorIndex(a)
}
