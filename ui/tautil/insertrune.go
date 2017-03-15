package tautil

func InsertRune(ta Texta, ru rune) {
	ta.EditOpen()
	if ta.SelectionOn() {
		// remove selection
		a, b := SelectionStringIndexes(ta)
		ta.EditDelete(a, b)
		ta.SetSelectionOn(false)
		ta.SetCursorIndex(a)
	}
	// insert
	s := string(ru)
	ta.EditInsert(ta.CursorIndex(), s)
	ta.EditClose()
	ta.SetCursorIndex(ta.CursorIndex() + len(s))
}
