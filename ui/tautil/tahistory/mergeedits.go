package tahistory

import (
	"container/list"
	"strings"
	"unicode"
)

func TryToMergeLastTwoEdits(h *History) {
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
