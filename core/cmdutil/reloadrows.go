package cmdutil

func ReloadRows(ed Editorer) {
	for _, erow := range ed.ERows() {
		ReloadRow(erow)
	}
}
func ReloadRow(erow ERower) {
	err := erow.ReloadContent()
	if err != nil {
		erow.Ed().Error(err)
	}
}

func ReloadRowsFiles(ed Editorer) {
	for _, erow := range ed.ERows() {
		_, fi, err := erow.FileInfo()
		if err != nil {
			// ed.Error(err) // TODO: would show error on special names
			continue
		}
		if fi.IsDir() {
			continue
		}
		ReloadRow(erow)
	}
}
