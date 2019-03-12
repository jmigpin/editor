package history

import (
	"container/list"
)

type History struct {
	l       *list.List
	cur     *list.Element // last edit, nil if list is empty
	maxSize int           // max elements in list // TODO: max data size
}

func NewHistory(maxSize int) *History {
	return &History{l: list.New(), maxSize: maxSize}
}

//----------

func (h *History) Append(edit *Edit) {
	if edit.Empty() {
		return
	}

	if h.cur == nil {
		h.cur = h.l.PushFront(edit)
	} else {
		h.cur = h.l.InsertAfter(edit, h.cur)
	}

	// remove all nexts
	for e := h.cur; e.Next() != nil; {
		h.l.Remove(e.Next())
	}

	// max size - clear backward
	if h.l.Len() > h.maxSize {
		diff := h.l.Len() - h.maxSize
		h.ClearOldN(diff)
	}

	// simplify history
	TryToMergeLastTwoEdits(h)
}

//----------

func (h *History) UndoRedo(redo bool) *Edit {
	if redo {
		return h.redo()
	} else {
		return h.undo()
	}
}

func (h *History) undo() *Edit {
	if h.cur == nil {
		return nil
	}
	edit := h.cur.Value.(*Edit)
	h.cur = h.cur.Prev()
	return edit
}

func (h *History) redo() *Edit {
	var next *list.Element
	if h.cur == nil {
		next = h.l.Front()
	} else {
		next = h.cur.Next()
	}
	if next == nil {
		return nil
	}
	h.cur = next
	return h.cur.Value.(*Edit)
}

//----------

func (h *History) Clear() {
	h.l = list.New()
	h.cur = nil
}

func (h *History) ClearForward() {
	for e := h.l.Back(); e != h.cur; e = h.l.Back() {
		h.l.Remove(e)
	}
}
func (h *History) ClearOldN(n int) {
	for e := h.l.Front(); e != h.cur && n > 0; e = h.l.Front() {
		h.l.Remove(e)
		n--
	}
}
