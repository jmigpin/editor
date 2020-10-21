package rwundo

import (
	"container/list"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/iout/iorw/rwedit"
)

////godebug:annotatefile

type Edits struct {
	list       list.List
	preCursor  rwedit.SimpleCursor
	postCursor rwedit.SimpleCursor
}

func (edits *Edits) Append(ur *UndoRedo) {
	// set pre cursor once
	if edits.list.Len() == 0 {
		if len(ur.D) > 0 {
			edits.preCursor.SetSelection(ur.Index, ur.Index+len(ur.D))
		} else {
			edits.preCursor.SetIndex(ur.Index)
		}
	}

	edits.list.PushBack(ur)

	// renew post cursor on each append
	if len(ur.I) > 0 {
		edits.postCursor.SetSelection(ur.Index, ur.Index+len(ur.I))
	} else {
		edits.postCursor.SetIndexSelectionOff(ur.Index)
	}
}

//----------

func (edits *Edits) MergeEdits(edits2 *Edits) {
	// append list
	for e := edits2.list.Front(); e != nil; e = e.Next() {
		ur := e.Value.(*UndoRedo)
		edits.Append(ur)
	}
	// merge cursor position
	if edits.list.Len() == 0 {
		edits.preCursor = edits2.preCursor
	}
	edits.postCursor = edits2.postCursor
}

//----------

func (edits *Edits) WriteUndoRedo(redo bool, w iorw.WriterAt) (rwedit.SimpleCursor, error) {
	if redo {
		for e := edits.list.Front(); e != nil; e = e.Next() {
			ur := e.Value.(*UndoRedo)
			if err := ur.Apply(redo, w); err != nil {
				return rwedit.SimpleCursor{}, err
			}
		}
		return edits.postCursor, nil
	} else {
		for e := edits.list.Back(); e != nil; e = e.Prev() {
			ur := e.Value.(*UndoRedo)
			if err := ur.Apply(redo, w); err != nil {
				return rwedit.SimpleCursor{}, err
			}
		}
		return edits.preCursor, nil
	}
}

//----------

func (edits *Edits) Entries() []*UndoRedo {
	w := make([]*UndoRedo, edits.list.Len())
	i := 0
	for e := edits.list.Front(); e != nil; e = e.Next() {
		ur := e.Value.(*UndoRedo)
		w[i] = ur
		i++
	}
	return w
}

func (edits *Edits) Empty() bool {
	for e := edits.list.Front(); e != nil; e = e.Next() {
		ur := e.Value.(*UndoRedo)
		if len(ur.D) > 0 || len(ur.I) > 0 {
			return false
		}
	}
	return true
}
