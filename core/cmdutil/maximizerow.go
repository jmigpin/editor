package cmdutil

func MaximizeRow(ed Editorer) {
	erow, ok := ed.ActiveERower()
	if !ok {
		ed.Errorf("no active row")
		return
	}
	erow.Row().Maximize()
}
