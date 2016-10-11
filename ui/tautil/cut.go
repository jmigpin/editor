package tautil

func Cut(ta Texta) {
	if !ta.SelectionOn() {
		return
	}
	a, b, ok := selectionStringIndexes(ta)
	if !ok {
		return
	}
	ta.SetClipboardString(ta.Text()[a:b])
	ta.SetText(ta.Text()[:a] + ta.Text()[b:]) // remove text
	ta.SetSelectionOn(false)
	ta.SetCursorIndex(a)
}
