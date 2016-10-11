package tautil

func InsertRune(ta Texta, ru rune) {
	text := ta.Text()
	if ta.SelectionOn() {
		// remove selection
		a, b, ok := selectionStringIndexes(ta)
		if !ok {
			return
		}
		text = text[:a] + text[b:] // remove text
		ta.SetSelectionOn(false)
		ta.SetCursorIndex(a)
	}
	// insert
	i := ta.CursorIndex()
	s := string(ru)
	text = text[:i] + s + text[i:]
	ta.SetText(text)
	ta.SetCursorIndex(ta.CursorIndex() + len(s))
}
