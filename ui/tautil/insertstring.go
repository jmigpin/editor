package tautil

func InsertString(ta Texta, s string) {
	ta.EditOpen()
	if ta.SelectionOn() {
		// remove selection
		a, b := SelectionStringIndexes(ta)
		ta.EditDelete(a, b)
		ta.SetSelectionOff()
		ta.SetCursorIndex(a)
	}
	// insert
	ta.EditInsert(ta.CursorIndex(), s)
	ta.EditClose()
	ta.SetCursorIndex(ta.CursorIndex() + len(s))
}
