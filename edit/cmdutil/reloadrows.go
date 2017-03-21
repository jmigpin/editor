package cmdutil

import (
	"os"

	"github.com/jmigpin/editor/ui"
)

func ReloadRows(ed Editorer) {
	for _, c := range ed.UI().Layout.Cols.Cols {
		for _, r := range c.Rows {
			ReloadRow(ed, r)
		}
	}
}
func ReloadRow(ed Editorer, row *ui.Row) {
	tsd := ed.RowToolbarStringData(row)
	p := tsd.FirstPartFilepath()
	content, err := ed.FilepathContent(p)
	if err != nil {
		ed.Error(err)
		return
	}
	row.TextArea.SetStrClear(content, false, false)
	ed.RowStatus().NotDirtyOrCold(row)
}

func ReloadRowsFiles(ed Editorer) {
	for _, c := range ed.UI().Layout.Cols.Cols {
		for _, r := range c.Rows {
			reloadRowFile(ed, r)
		}
	}
}
func reloadRowFile(ed Editorer, row *ui.Row) {
	tsd := ed.RowToolbarStringData(row)
	p := tsd.FirstPartFilepath()
	// check if its a file
	fi, err := os.Stat(p)
	if err != nil {
		return
	}
	if fi.IsDir() {
		return
	}

	ReloadRow(ed, row)
}
