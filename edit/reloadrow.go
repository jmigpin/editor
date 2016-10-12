package edit

import "github.com/jmigpin/editor/ui"

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
	row.TextArea.SetText(content)
	row.TextArea.SetSelectionOn(false)
	row.Square.SetDirty(false)
	row.Square.SetCold(false)
}
