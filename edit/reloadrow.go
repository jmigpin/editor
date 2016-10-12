package edit

import "jmigpin/editor/ui"

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
	err := loadRowContent(ed, row)
	if err != nil && !tolerant {
		ed.Error(err)
	}
}
