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

	// Keep position to keep seeing the content in the same place.
	// Works well when a reload happens and its identical to the previous content.
	ta := row.TextArea
	ci := ta.CursorIndex()
	oy := ta.OffsetY()
	// clear str
	ta.ClearStr(content)
	// restore position
	ta.SetCursorIndex(ci)
	ta.SetOffsetY(oy)

	row.Square.SetDirty(false)
	row.Square.SetCold(false)
}
