package cmdutil

func ReloadRows(ed Editorer) {
	for _, erow := range ed.ERows() {
		ReloadRow(erow)
	}
}
func ReloadRow(erow ERower) {
	err := erow.ReloadContent()
	if err != nil {
		erow.Editorer().Error(err)
	}
}

func ReloadRowsFiles(ed Editorer) {
	for _, erow := range ed.ERows() {
		reloadRowFile(erow)
	}
}
func reloadRowFile(erow ERower) {
	_, fi, ok := erow.FileInfo()
	if !ok {
		return
	}
	if fi.IsDir() {
		return
	}
	ReloadRow(erow)
}
