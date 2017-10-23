package tahistory

import (
	"container/list"
)

type History struct {
	l       *list.List
	cur     *list.Element // last edit, nil if list is empty
	maxSize int
}

func NewHistory(maxSize int) *History {
	return &History{l: list.New(), maxSize: maxSize}
}
func (h *History) PushEdit(edit *Edit) {
	if h.cur == nil {
		h.cur = h.l.PushFront(edit)
	} else {
		h.cur = h.l.InsertAfter(edit, h.cur)
	}

	// remove all nexts
	for e := h.cur; e.Next() != nil; {
		h.l.Remove(e.Next())
	}

	// max size
	if h.l.Len() > h.maxSize {
		diff := h.l.Len() - h.maxSize
		for i := 0; i < diff; i++ {
			if h.l.Front() == h.cur {
				panic("!")
			}
			h.l.Remove(h.l.Front())
		}
	}
}
func (h *History) Undo(str string) (string, int, bool) {
	if h.cur == nil {
		return "", 0, false
	}
	edit := h.cur.Value.(*Edit)
	h.cur = h.cur.Prev()
	s, i := edit.ApplyUndo(str)
	return s, i, true
}
func (h *History) Redo(str string) (string, int, bool) {
	var next *list.Element
	if h.cur == nil {
		next = h.l.Front()
	} else {
		next = h.cur.Next()
	}
	if next == nil {
		return "", 0, false
	}
	h.cur = next
	edit := h.cur.Value.(*Edit)
	s, i := edit.Apply(str)
	return s, i, true
}

func (h *History) Clear() {
	h.l = list.New()
	h.cur = nil
}
