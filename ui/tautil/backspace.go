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
		_, a, ok = PreviousRuneIndex(ta.Text(), b)
		if !ok {
			return
		}
	}
	ta.SetText(ta.Text()[:a] + ta.Text()[b:]) // remove text
	ta.SetCursorIndex(a)
}
