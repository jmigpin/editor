package cmdutil

import "github.com/jmigpin/editor/ui"

type ReopenRow struct {
	ed Editorer
	q  []*RowState
}

func NewReopenRow(ed Editorer) *ReopenRow {
	return &ReopenRow{ed: ed}
}
func (reop *ReopenRow) Add(row *ui.Row) {
	state := NewRowState(row)
	reop.q = append(reop.q, state)

	// limit q entries
	max := 5
	l := len(reop.q)
	if l > max {
		reop.q = append([]*RowState{}, reop.q[l-max:]...)
	}
}
func (reop *ReopenRow) Reopen() bool {
	if len(reop.q) == 0 {
		return false
	}
	l := len(reop.q)
	state := reop.q[l-1]
	reop.q = reop.q[:l-1] // remove from q

	col, nextRow := reop.ed.GoodColumnRowPlace()
	erow := NewERowFromRowState(reop.ed, state, col, nextRow)
	erow.Row().WarpPointer()
	return true
}
