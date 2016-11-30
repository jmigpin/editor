package edit

import (
	"os"

	"github.com/jmigpin/editor/ui"
)

func reloadRows(ed *Editor) {
	for _, c := range ed.ui.Layout.Cols.Cols {
		for _, r := range c.Rows {
			reloadRow2(ed, r, true)
		}
	}
}
func reloadRow(ed *Editor, row *ui.Row) {
	reloadRow2(ed, row, false)
}
func reloadRow2(ed *Editor, row *ui.Row, tolerant bool) {
	tsd := ed.rowToolbarStringData(row)
	p := tsd.FirstPartFilepath()
	content, err := filepathContent(p)
	if err != nil {
		if !tolerant {
			ed.Error(err)
			return
		}
	}
	row.TextArea.ClearStr(content, true)
	row.Square.SetDirty(false)
	row.Square.SetCold(false)
}

func reloadRowsFiles(ed *Editor) {
	for _, c := range ed.ui.Layout.Cols.Cols {
		for _, r := range c.Rows {
			reloadRowFile(ed, r)
		}
	}
}
func reloadRowFile(ed *Editor, row *ui.Row) {
	tsd := ed.rowToolbarStringData(row)
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
