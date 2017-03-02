package tautil

func InsertRune(ta Texta, ru rune) {
	if ta.SelectionOn() {
		// remove selection
		a, b := SelectionStringIndexes(ta)
		ta.EditRemove(a, b)
		ta.SetSelectionOn(false)
		ta.SetCursorIndex(a)
	}
	// insert
	s := string(ru)
	ta.EditInsert(ta.CursorIndex(), s)
	ta.EditDone()
	ta.SetCursorIndex(ta.CursorIndex() + len(s))
}
