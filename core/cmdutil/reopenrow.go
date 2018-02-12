package cmdutil

import (
	"container/list"

	"github.com/jmigpin/editor/ui"
)

// TODO: rename to rowreopener
type ReopenRow struct {
	ed Editorer
	q  list.List
}

func NewReopenRow(ed Editorer) *ReopenRow {
	return &ReopenRow{ed: ed}
}
func (rr *ReopenRow) Add(row *ui.Row) {
	state := NewRowState(row)

	rr.q.PushBack(state)

	// limit entries
	max := 5
	for rr.q.Len() > max {
		rr.q.Remove(rr.q.Front())
	}
}
func (rr *ReopenRow) Reopen() bool {
	if rr.q.Len() == 0 {
		return false
	}

	// pop state from queue
	state := rr.q.Remove(rr.q.Back()).(*RowState)

	col, nextRow := rr.ed.GoodColumnRowPlace()
	erow := NewERowFromRowState(rr.ed, state, col, nextRow)
	erow.Flash()
	return true
}
