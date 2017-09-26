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
		if erow.IsSpecialName() || erow.IsDir() {
			continue
		}
		ReloadRow(erow)
	}
}
