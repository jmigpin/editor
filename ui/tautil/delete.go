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
		_, b, ok = NextRuneIndex(ta.Text(), a)
		if !ok {
			return
		}
	}
	ta.SetText(ta.Text()[:a] + ta.Text()[b:]) // remove text
	ta.SetCursorIndex(a)
}
