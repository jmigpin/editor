package tahistory

import (
	"container/list"
	"strings"
	"unicode"
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

func (h *History) TryToMergeLastTwoEdits() {
	if h.cur == nil {
		return
	}
	prev := h.cur.Prev()
	if prev == nil {
		return
	}
	ed1 := prev.Value.(*Edit)  // oldest
	ed2 := h.cur.Value.(*Edit) // recent

	insConsecutiveLetters := func() bool {
		elemOk := func(id *InsertDelete) bool {
			return id.IsInsert && strings.TrimFunc(id.str, unicode.IsLetter) == ""
		}
		consecOk := func(e1, e2 *list.Element) bool {
			v1 := e1.Value.(*InsertDelete)
			v2 := e2.Value.(*InsertDelete)
			if !(elemOk(v1) && elemOk(v2)) {
				return false
			}
			return v1.index == v2.index-1
		}
		prev := ed1.Front()
		for e := prev.Next(); e != nil; e = e.Next() {
			if !consecOk(prev, e) {
				return false
			}
			prev = e
		}
		for e := ed2.Front(); e != nil; e = e.Next() {
			if !consecOk(prev, e) {
				return false
			}
			prev = e
		}
		return true
	}

	consecutiveSpaces := func() bool {
		if ed2.Len() > 1 {
			return false
		}
		elemOk := func(id *InsertDelete) bool {
			// can be del or insert
			return strings.TrimSpace(id.str) == ""
		}
		consecOk := func(e1, e2 *list.Element) bool {
			v1 := e1.Value.(*InsertDelete)
			v2 := e2.Value.(*InsertDelete)
			if !(elemOk(v1) && elemOk(v2)) {
				return false
			}
			// del, backspace, space/newline
			return v1.index == v2.index ||
				v1.index-len(v1.str) == v2.index ||
				v1.index+len(v1.str) == v2.index
		}
		prev := ed1.Front()
		for e := prev.Next(); e != nil; e = e.Next() {
			if !consecOk(prev, e) {
				return false
			}
			prev = e
		}
		for e := ed2.Front(); e != nil; e = e.Next() {
			if !consecOk(prev, e) {
				return false
			}
			prev = e
		}
		return true
	}

	if !(insConsecutiveLetters() || consecutiveSpaces()) {
		return
	}

	// merge ed2 into ed1
	ed1.PushBackList(&ed2.List)

	// remove ed2 (h.cur)
	h.l.Remove(h.cur)
	h.cur = prev
}
