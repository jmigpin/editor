package history

import (
	"bytes"
	"unicode"

	"github.com/jmigpin/editor/util/iout"
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

	if false ||
		insertConsecutiveLetters(ed1, ed2) ||
		consecutiveSpaces(ed1, ed2) {

		// merge ed2 into ed1
		ed1.list.PushBackList(&ed2.list)
		ed1.PostState = ed2.PostState

		// remove ed2 (h.cur)
		h.l.Remove(h.cur)
		h.cur = prev
	}
}

//----------

func insertConsecutiveLetters(ed1, ed2 *Edit) bool {
	urs1 := ed1.Entries()
	prev := urs1[0]
	if !urIsLetterInsert(prev) {
		return false
	}
	for i := 1; i < len(urs1); i++ {
		e := urs1[i]
		if !urIsLetterInsert(e) {
			return false
		}
		if !urConsecutive(prev, e) {
			return false
		}
		prev = e
	}
	urs2 := ed2.Entries()
	for i := 0; i < len(urs2); i++ {
		e := urs2[i]
		if !urIsLetterInsert(e) {
			return false
		}
		if !urConsecutive(prev, e) {
			return false
		}
		prev = e
	}
	return true
}

//----------

func consecutiveSpaces(ed1, ed2 *Edit) bool {
	urs1 := ed1.Entries()
	prev := urs1[0]
	if !urIsSpace(prev) {
		return false
	}
	for i := 1; i < len(urs1); i++ {
		e := urs1[i]
		if !urIsSpace(e) {
			return false
		}
		if !urConsecutiveEitherSide(prev, e) {
			return false
		}
		prev = e
	}
	urs2 := ed2.Entries()
	for i := 0; i < len(urs2); i++ {
		e := urs2[i]
		if !urIsSpace(e) {
			return false
		}
		if !urConsecutiveEitherSide(prev, e) {
			return false
		}
		prev = e
	}
	return true
}

//----------

func urIsLetterInsert(ur *iout.UndoRedo) bool {
	r := []rune(string(ur.S))
	// not-insert is the undo of an "insert"
	return !ur.Insert && len(r) == 1 && unicode.IsLetter(r[0])
}

func urIsSpace(ur *iout.UndoRedo) bool {
	// can be insert or delete
	return len(bytes.TrimSpace(ur.S)) == 0
}

func urConsecutive(ur1, ur2 *iout.UndoRedo) bool {
	return ur1.Index+len(ur1.S) == ur2.Index
}

func urConsecutiveEitherSide(ur1, ur2 *iout.UndoRedo) bool {
	return ur1.Index+len(ur1.S) == ur2.Index || // moved to the right
		ur1.Index == ur2.Index+len(ur2.S) || // moved to the left
		ur1.Index == ur2.Index // stayed in place
}
