package rwundo

import (
	"container/list"
	"fmt"
	"sync"

	"github.com/jmigpin/editor/v2/util/iout/iorw/rwedit"
)

////godebug:annotatefile

type History struct {
	maxLen int // max elements in list // TODO: max data size
	hlist  *HList
	ugroup struct { // undo group
		sync.Mutex
		ohlist *HList // original list
		c      rwedit.SimpleCursor
	}
}

func NewHistory(maxLen int) *History {
	h := &History{hlist: NewHList(), maxLen: maxLen}
	return h
}

//----------

func (h *History) Append(edits *Edits) { h.hlist.Append(edits, h.maxLen) }
func (h *History) Clear()              { h.hlist.Clear() }
func (h *History) ClearUndones()       { h.hlist.ClearUndones() }

//func (h *History) MergeNDoneBack(n int) { h.hlist.MergeNDoneBack(n) }

//----------

func (h *History) UndoRedo(redo, peek bool) (*Edits, bool) {
	// the call to undo could be inside an undogroup, use the original list; usually this is ok since the only operations should be undo/redo, but if other write operations are done while on this undogroup, there could be undefined behaviour (programmer responsability)
	h.ugroup.Lock()
	defer h.ugroup.Unlock()
	hl := h.hlist
	if h.ugroup.ohlist != nil {
		hl = h.ugroup.ohlist
	}

	if redo {
		return hl.Redo(peek)
	} else {
		return hl.Undo(peek)
	}
}

//----------

func (h *History) BeginUndoGroup(c rwedit.SimpleCursor) {
	h.ugroup.Lock()
	defer h.ugroup.Unlock()
	if h.ugroup.ohlist != nil {
		panic("history undo group already set")
	}

	// replace hlist
	h.ugroup.ohlist = h.hlist
	h.hlist = NewHList()

	// keep cursordata
	h.ugroup.c = c
}

func (h *History) EndUndoGroup(c rwedit.SimpleCursor) {
	h.ugroup.Lock()
	defer h.ugroup.Unlock()
	if h.ugroup.ohlist == nil {
		panic("history undo group is not set")
	}
	defer func() { h.ugroup.ohlist = nil }()

	// merge all, should then have either 0 or 1 element
	h.hlist.mergeToDoneBack(h.hlist.list.Front())
	if h.hlist.list.Len() > 1 {
		panic(fmt.Sprintf("history undo group merge: %v", h.hlist.list.Len()))
	}

	if h.hlist.list.Len() == 1 {
		// overwrite undogroup cursors - allows a setbytes to not end with the full content selected since it overwrites all
		edits := h.hlist.list.Front().Value.(*Edits)
		edits.preCursor = h.ugroup.c
		edits.postCursor = c
		// append undogroup elements to the original list
		h.ugroup.ohlist.Append(edits, h.maxLen)
	}

	// restore original list
	h.hlist = h.ugroup.ohlist
}

//----------

type HList struct {
	list   *list.List
	undone *list.Element
}

func NewHList() *HList {
	return &HList{list: list.New()}
}

//----------

func (hl *HList) DoneBack() *list.Element {
	if hl.undone != nil {
		return hl.undone.Prev()
	}
	return hl.list.Back()
}

//----------

func (hl *HList) Append(edits *Edits, maxLen int) {
	if edits.Empty() {
		return
	}
	hl.ClearUndones()       // make back clear
	hl.list.PushBack(edits) // add to the back
	hl.clearOlds(maxLen)
	tryToMergeLastTwoEdits(hl) // simplify history
}

//----------

func (hl *HList) Undo(peek bool) (*Edits, bool) {
	u := hl.DoneBack()
	if u == nil {
		return nil, false
	}
	if !peek {
		hl.undone = u
	}
	return u.Value.(*Edits), true
}

func (hl *HList) Redo(peek bool) (*Edits, bool) {
	u := hl.undone
	if u == nil {
		return nil, false
	}
	if !peek {
		hl.undone = hl.undone.Next()
	}
	return u.Value.(*Edits), true
}

//----------

func (hl *HList) Clear() {
	hl.list = list.New()
	hl.undone = nil
}

func (hl *HList) ClearUndones() {
	for e := hl.undone; e != nil; {
		u := e.Next()
		hl.list.Remove(e)
		e = u
	}
	hl.undone = nil
}

func (hl *HList) clearOlds(maxLen int) {
	for hl.list.Len() > maxLen {
		e := hl.list.Front()
		if e == hl.undone {
			break
		}
		hl.list.Remove(e)
	}
}

//----------

func (hl *HList) mergeToDoneBack(elem *list.Element) {
	for hl.mergeNextNotUndone(elem) {
	}
}
func (hl *HList) mergeNextNotUndone(elem *list.Element) bool {
	if elem == nil {
		return false
	}
	if elem == hl.undone {
		return false
	}
	edits := elem.Value.(*Edits)
	next := elem.Next()
	if next == nil {
		return false
	}
	nextEdits := next.Value.(*Edits)
	edits.MergeEdits(nextEdits)
	hl.list.Remove(next)
	return true
}

//func (hl *HList) MergeNDoneBack(n int) {
//	e := hl.DoneBack()
//	for ; n > 0 && e != nil; e = e.Prev() {
//		n--
//	}
//	hl.mergeToDoneBack(e)
//}

//----------

func (hl *HList) NDoneBack(n int) ([]*Edits, []*list.Element) {
	b := hl.DoneBack()
	w := []*Edits{}
	u := []*list.Element{}
	for e := b; e != nil && n > 0; e = e.Prev() {
		edits := e.Value.(*Edits)
		w = append(w, edits)
		u = append(u, e)
		n--
	}
	// reverse append order
	l := len(w)
	for i := 0; i < l/2; i++ {
		k := l - 1 - i
		w[i], w[k] = w[k], w[i]
		u[i], u[k] = u[k], u[i]
	}
	return w, u
}
