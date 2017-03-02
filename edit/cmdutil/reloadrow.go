package cmdutil

import (
	"os"

	"github.com/jmigpin/editor/ui"
)

func ReloadRows(ed Editori) {
	for _, c := range ed.UI().Layout.Cols.Cols {
		for _, r := range c.Rows {
			reloadRow2(ed, r, true)
		}
	}
}
func ReloadRow(ed Editori, row *ui.Row) {
	reloadRow2(ed, row, false)
}
func reloadRow2(ed Editori, row *ui.Row, tolerant bool) {
	tsd := ed.RowToolbarStringData(row)
	p := tsd.FirstPartFilepath()
	content, err := ed.FilepathContent(p)
	if err != nil {
		if !tolerant {
			ed.Error(err)
			return
		}
	}
	row.TextArea.SetStrClear2(content, false, false)
	row.Square.SetDirty(false)
	row.Square.SetCold(false)
}

func ReloadRowsFiles(ed Editori) {
	for _, c := range ed.UI().Layout.Cols.Cols {
		for _, r := range c.Rows {
			reloadRowFile(ed, r)
		}
	}
}
func reloadRowFile(ed Editori, row *ui.Row) {
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
	// reload content
	reloadRow2(ed, row, true)
}
