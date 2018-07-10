package core

import (
	"container/list"

	"github.com/jmigpin/editor/ui"
)

type RowReopener struct {
	ed *Editor
	q  list.List
}

func NewRowReopener(ed *Editor) *RowReopener {
	return &RowReopener{ed: ed}
}
func (rr *RowReopener) Add(row *ui.Row) {
	rstate := NewRowState(row)

	rr.q.PushBack(rstate)

	// limit entries
	max := 5
	for rr.q.Len() > max {
		rr.q.Remove(rr.q.Front())
	}
}
func (rr *RowReopener) Reopen() {
	if rr.q.Len() == 0 {
		rr.ed.Errorf("no rows to reopen")
		return
	}

	// pop state from queue
	rstate := rr.q.Remove(rr.q.Back()).(*RowState)

	rowPos := rr.ed.GoodRowPos()
	erow, ok, err := rstate.OpenERow(rr.ed, rowPos)
	if err != nil {
		rr.ed.Error(err)
	}
	if !ok {
		return
	}
	rstate.RestorePos(erow)
	erow.Flash()
}
