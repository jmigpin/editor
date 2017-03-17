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
func NewRowFromRowState(ed Editorer, state *RowState, col *ui.Column) *ui.Row {
	// create row
	row := ed.NewRow(col)
	row.Toolbar.SetStrClear(state.TbStr, true, true)
	row.Toolbar.SetCursorIndex(state.TbCursorIndex)

	// content
	tsd := ed.RowToolbarStringData(row)
	p := tsd.FirstPartFilepath()
	content, err := ed.FilepathContent(p)
	if err != nil {
		ed.Error(err)
		return row
	}

	row.TextArea.SetStrClear(content, true, true)
	row.Square.SetDirty(false)
	row.TextArea.SetCursorIndex(state.TaCursorIndex)
	row.TextArea.SetOffsetIndex(state.TaOffsetIndex)
	return row
}
