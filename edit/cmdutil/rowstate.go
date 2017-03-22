package cmdutil

import "github.com/jmigpin/editor/ui"

// Used in sessions and reopenrow.
type RowState struct {
	TbStr         string
	TbCursorIndex int
	TaCursorIndex int
	TaOffsetIndex int
}

func NewRowState(row *ui.Row) *RowState {
	return &RowState{
		TbStr:         row.Toolbar.Str(),
		TbCursorIndex: row.Toolbar.CursorIndex(),
		TaCursorIndex: row.TextArea.CursorIndex(),
		TaOffsetIndex: row.TextArea.OffsetIndex(),
	}
}
func NewERowFromRowState(ed Editorer, state *RowState, col *ui.Column, rowIndex int) ERower {
	erow := ed.NewERow(state.TbStr, col, rowIndex)
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
