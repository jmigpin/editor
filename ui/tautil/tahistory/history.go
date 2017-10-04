package tahistory

import (
	"container/list"
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
func (h *History) PushStrEdit(edit *StrEdit) {
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

	h.tryToMergeLastTwoEdits()
}
func (h *History) Undo(str string) (string, int, bool) {
	if h.cur == nil {
		return "", 0, false
	}
	edit := h.cur.Value.(*StrEdit)
	h.cur = h.cur.Prev()
	s, i := edit.undos.Apply(str)
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
	edit := h.cur.Value.(*StrEdit)
	s, i := edit.edits.Apply(str)
	return s, i, true
}
func (h *History) Clear() {
	h.l = list.New()
	h.cur = nil
}

func (h *History) tryToMergeLastTwoEdits() {
	if h.cur == nil {
		return
	}
	prev := h.cur.Prev()
	if prev == nil {
		return
	}

	se1 := prev.Value.(*StrEdit)  // oldest
	se2 := h.cur.Value.(*StrEdit) // recent

	editIsInsertLetter := func(edit *StrEdit) (int, bool) {
		if edit.edits.Len() != 1 {
			return 0, false
		}
		sei, ok := edit.edits.Back().Value.(*StrEditInsert)
		if !ok {
			return 0, false
		}
		if len(sei.str) != 1 || !unicode.IsLetter(rune(sei.str[0])) {
			return 0, false
		}
		return sei.index, true
	}

	editLastActionIsInsertLetter := func(edit *StrEdit) (int, bool) {
		if edit.edits.Len() == 0 {
			return 0, false
		}

		// all inserts
		for e := edit.edits.Front(); e != nil; e = e.Next() {
			_, ok := e.Value.(*StrEditInsert)
			if !ok {
				return 0, false
			}
		}

		sei, ok := edit.edits.Back().Value.(*StrEditInsert)
		if !ok {
			return 0, false
		}
		if len(sei.str) != 1 || !unicode.IsLetter(rune(sei.str[0])) {
			return 0, false
		}
		return sei.index, true
	}

	editIsDeleteOne := func(edit *StrEdit) (int, bool) {
		if edit.edits.Len() != 1 {
			return 0, false
		}
		sed, ok := edit.edits.Back().Value.(*StrEditDelete)
		if !ok {
			return 0, false
		}
		return sed.index, sed.index2-sed.index == 1
	}

	editLastActionIsDeleteOne := func(edit *StrEdit) (int, bool) {
		if edit.edits.Len() == 0 {
			return 0, false
		}
		sed, ok := edit.edits.Back().Value.(*StrEditDelete)
		if !ok {
			return 0, false
		}
		return sed.index, sed.index2-sed.index == 1
	}

	insertedConsecutiveLetters := func() bool {
		a1, ok := editLastActionIsInsertLetter(se1)
		if !ok {
			return false
		}
		a2, ok := editIsInsertLetter(se2)
		if !ok {
			return false
		}
		if a1 != a2-1 { // size 1
			return false
		}
		return true
	}

	deletedConsecutiveBackspaces := func() bool {
		a1, ok := editLastActionIsDeleteOne(se1)
		if !ok {
			return false
		}
		a2, ok := editIsDeleteOne(se2)
		if !ok {
			return false
		}
		if a1 != a2+1 {
			return false
		}
		return true
	}

	if !(insertedConsecutiveLetters() ||
		deletedConsecutiveBackspaces()) {
		return
	}

	// merge s2 into s1
	se1.edits.PushBackList(&se2.edits.List)
	se1.undos.PushFrontList(&se2.undos.List)

	// remove se2 (h.cur)
	h.l.Remove(h.cur)
	h.cur = prev
}
