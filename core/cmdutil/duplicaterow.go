package cmdutil

func DuplicateRow(ed Editorer, erow ERower) {
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
	next := erow.Row().NextRow()

	// make duplicate (have same filename)
	filename := erow.Filename()
	tbStr := filename
	erow2 := ed.NewERowerBeforeRow(tbStr, col, next)
	err := erow2.LoadContentClear()
	if err != nil {
		ed.Error(err)
		return
	}

	// set position
	ta := erow.Row().TextArea
	ta2 := erow2.Row().TextArea
	ta2.SetCursorIndex(ta.CursorIndex())
	ta2.MakeCursorVisible()

	erow2.Flash()
}
