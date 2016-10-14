package tautil

func Delete(ta Texta) {
	var a, b int
	var ok bool
	if ta.SelectionOn() {
		a, b, ok = selectionStringIndexes(ta)
		if !ok {
			return
		}
		ta.SetSelectionOn(false)
	} else {
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
