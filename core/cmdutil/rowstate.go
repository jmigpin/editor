package cmdutil

import "github.com/jmigpin/editor/ui"

// Used in sessions and reopenrow.
type RowState struct {
	TbStr         string
	TbCursorIndex int
	TaCursorIndex int
	TaOffsetIndex int
	EndPercent    float64
}

func NewRowState(row *ui.Row) *RowState {
	rs := &RowState{
		TbStr:         row.Toolbar.Str(),
		TbCursorIndex: row.Toolbar.CursorIndex(),
		TaCursorIndex: row.TextArea.CursorIndex(),
		TaOffsetIndex: row.TextArea.OffsetIndex(),
	}
	// Keep end percent if possible. ReopenRow uses this after the row is removed and doesn't have a parent anymore.
	if row.Parent() != nil {
		rs.EndPercent = row.Col.RowsLayout.ChildEndPercent(row)
	}
	return rs
}
func NewERowFromRowState(ed Editorer, state *RowState, col *ui.Column, nextRow *ui.Row) ERower {
	erow := ed.NewERowBeforeRow(state.TbStr, col, nextRow)
	row := erow.Row()
	row.Toolbar.SetCursorIndex(state.TbCursorIndex)
	err := erow.LoadContentClear()
	if err != nil {
		ed.Error(err)
		return erow
	}
	row.TextArea.SetCursorIndex(state.TaCursorIndex)
	row.TextArea.SetOffsetIndex(state.TaOffsetIndex)
	return erow
}
