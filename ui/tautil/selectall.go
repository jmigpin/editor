package tautil

func SelectAll(ta Texta) {
	ta.SetSelectionOn(true)
	ta.SetSelectionIndex(0)
	ta.SetCursorIndex(len(ta.Str()))
}
