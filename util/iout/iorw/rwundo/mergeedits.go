package rwundo

import (
	"bytes"
	"unicode"
)

////godebug:annotatefile

func tryToMergeLastTwoEdits(hl *HList) {
	editsL, elemsL := hl.NDoneBack(2)
	if len(editsL) != 2 {
		return
	}
	if insertConsecutiveLetters(editsL[0], editsL[1]) ||
		consecutiveSpaces(editsL[0], editsL[1]) {
		hl.mergeToDoneBack(elemsL[0])
	}
}

//----------

func insertConsecutiveLetters(ed1, ed2 *Edits) bool {
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

func consecutiveSpaces(ed1, ed2 *Edits) bool {
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

func urIsLetterInsert(ur *UndoRedo) bool {
	if ur.IsInsertOnly() {
		r := []rune(string(ur.I))
		return len(r) == 1 && unicode.IsLetter(r[0])
	}
	return false
}

func urIsSpace(ur *UndoRedo) bool {
	return len(bytes.TrimSpace(ur.D)) == 0 && len(bytes.TrimSpace(ur.I)) == 0
}

func urConsecutive(ur1, ur2 *UndoRedo) bool {
	return ur1.Index+len(ur1.I) == ur2.Index
}

func urConsecutiveEitherSide(ur1, ur2 *UndoRedo) bool {
	return ur1.Index+len(ur1.I) == ur2.Index || // moved to the right
		ur1.Index == ur2.Index+len(ur2.I) || // moved to the left
		ur1.Index == ur2.Index // stayed in place
}
