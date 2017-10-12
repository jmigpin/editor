package cmdutil

func DuplicateRow(ed Editorer) {
	erow, ok := ed.ActiveERow()
	if !ok {
		return
	}

	if erow.IsDir() {
		ed.Errorf("can't duplicate directory: %s", erow.Filename())
		return
	}
	if erow.IsSpecialName() {
		ed.Errorf("can't duplicate special name: %s", erow.Name())
		return
	}

	// col/row position of the duplicate
	col := erow.Row().Col
	next, ok := erow.Row().NextRow()
	if !ok {
		next = nil
	}

	// make duplicate (have same filename)
	filename := erow.Filename()
	tbStr := filename
	erow2 := ed.NewERowBeforeRow(tbStr, col, next)

	erow.UpdateState() // visual cue for duplicates
	erow.UpdateDuplicates()

	// set position
	ta := erow.Row().TextArea
	ta2 := erow2.Row().TextArea
	ta2.SetCursorIndex(ta.CursorIndex())

	erow2.Row().WarpPointer()
}
