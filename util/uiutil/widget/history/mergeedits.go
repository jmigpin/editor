package history

import (
	"bytes"
	"container/list"
	"unicode"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func TryToMergeLastTwoEdits(h *History) {
	tce, ok := LastTwoEdits(h)
	if !ok {
		return
	}
	if insertConsecutiveLetters(tce.Ed1, tce.Ed2) ||
		consecutiveSpaces(tce.Ed1, tce.Ed2) {
		MergeTwoEdits(h, tce)
	}
}

func MergeLastTwoEdits(h *History) {
	tce, ok := LastTwoEdits(h)
	if !ok {
		return
	}
	MergeTwoEdits(h, tce)
}

//----------

func MergeTwoEdits(h *History, tce *TwoConsecutiveEdits) {
	// merge ed2 into ed1
	tce.Ed1.list.PushBackList(&tce.Ed2.list)
	tce.Ed1.PostState = tce.Ed2.PostState

	// remove ed2
	h.l.Remove(*tce.Elem2)
	*tce.Elem2 = *tce.Elem1 // elem2 usually points to h.cur
}

//----------

type TwoConsecutiveEdits struct {
	Ed1, Ed2     *Edit
	Elem1, Elem2 **list.Element
}

func LastTwoEdits(h *History) (*TwoConsecutiveEdits, bool) {
	if h.cur == nil {
		return nil, false
	}
	prev := h.cur.Prev()
	if prev == nil {
		return nil, false
	}
	ed1 := prev.Value.(*Edit)  // oldest
	ed2 := h.cur.Value.(*Edit) // recent
	tce := &TwoConsecutiveEdits{ed1, ed2, &prev, &h.cur}
	return tce, true
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

func urIsLetterInsert(ur *iorw.UndoRedo) bool {
	r := []rune(string(ur.S))
	// not-insert is the undo of an "insert"
	return !ur.Insert && len(r) == 1 && unicode.IsLetter(r[0])
}

func urIsSpace(ur *iorw.UndoRedo) bool {
	// can be insert or delete
	return len(bytes.TrimSpace(ur.S)) == 0
}

func urConsecutive(ur1, ur2 *iorw.UndoRedo) bool {
	return ur1.Index+len(ur1.S) == ur2.Index
}

func urConsecutiveEitherSide(ur1, ur2 *iorw.UndoRedo) bool {
	return ur1.Index+len(ur1.S) == ur2.Index || // moved to the right
		ur1.Index == ur2.Index+len(ur2.S) || // moved to the left
		ur1.Index == ur2.Index // stayed in place
}
