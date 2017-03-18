package tautil

import "unicode"

type EditHistory struct {
	q               []*StrEdit // edits queue
	start, end, cur int        // q positions, start/end are always positive
}

func NewEditHistory(n int) *EditHistory {
	return &EditHistory{q: make([]*StrEdit, n)}
}
func (h *EditHistory) qmod(index int) **StrEdit {
	return &h.q[index%len(h.q)]
}
func (h *EditHistory) PushEdit(edit *StrEdit) {
	*h.qmod(h.cur) = edit
	h.cur++
	h.end = h.cur
	if h.end-h.start > len(h.q) {
		h.start = h.end - len(h.q)
	}

	h.tryToMergeLastTwoEdits()
}
func (h *EditHistory) PopUndo(str string) (string, int, bool) {
	if h.cur-1 < h.start {
		return "", 0, false // no undos
	}
	h.cur--
	edit := *h.qmod(h.cur)
	s, i := edit.undos.Apply(str)
	return s, i, true
}
func (h *EditHistory) UnpopRedo(str string) (string, int, bool) {
	if h.cur == h.end {
		return "", 0, false // no redos
	}
	edit := *h.qmod(h.cur)
	h.cur++
	s, i := edit.edits.Apply(str)
	return s, i, true
}
func (h *EditHistory) ClearQ() {
	h.start, h.cur, h.end = 0, 0, 0
	for i := range h.q {
		h.q[i] = nil
	}
}
func (h *EditHistory) tryToMergeLastTwoEdits() {
	e1 := h.cur - 2
	e2 := h.cur - 1
	if e1 < h.start || e2 < h.start {
		return
	}
	edit1 := *h.qmod(e1)
	edit2 := *h.qmod(e2)

	editIsInsertLetter := func(edit *StrEdit) (int, bool) {
		if len(edit.edits) != 1 {
			return 0, false
		}
		sei, ok := edit.edits[0].(*StrEditInsert)
		if !ok {
			return 0, false
		}
		if len(sei.str) != 1 || !unicode.IsLetter(rune(sei.str[0])) {
			return 0, false
		}
		return sei.index, true
	}

	editLastActionIsInsertLetter := func(edit *StrEdit) (int, bool) {
		// all inserts
		for _, u := range edit.edits {
			_, ok := u.(*StrEditInsert)
			if !ok {
				return 0, false
			}
		}

		l := len(edit.edits)
		sei, ok := edit.edits[l-1].(*StrEditInsert)
		if !ok {
			return 0, false
		}
		if len(sei.str) != 1 || !unicode.IsLetter(rune(sei.str[0])) {
			return 0, false
		}
		return sei.index, true
	}

	editIsDeleteOne := func(edit *StrEdit) (int, bool) {
		if len(edit.edits) != 1 {
			return 0, false
		}
		sed, ok := edit.edits[0].(*StrEditDelete)
		if !ok {
			return 0, false
		}
		return sed.index, sed.index2-sed.index == 1
	}

	editLastActionIsDeleteOne := func(edit *StrEdit) (int, bool) {
		l := len(edit.edits)
		sed, ok := edit.edits[l-1].(*StrEditDelete)
		if !ok {
			return 0, false
		}
		return sed.index, sed.index2-sed.index == 1
	}

	insertedConsecutiveLetters := func() bool {
		a1, ok := editLastActionIsInsertLetter(edit1)
		if !ok {
			return false
		}
		a2, ok := editIsInsertLetter(edit2)
		if !ok {
			return false
		}
		if a1 != a2-1 { // size 1
			return false
		}
		return true
	}

	deletedConsecutiveBackspaces := func() bool {
		a1, ok := editLastActionIsDeleteOne(edit1)
		if !ok {
			return false
		}
		a2, ok := editIsDeleteOne(edit2)
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

	// merge: add edit2 edits to edit1
	edit1.edits = append(edit1.edits, edit2.edits...)
	edit1.undos = append(edit2.undos, edit1.undos...)

	// remove edit2 from q
	// ok to remove like this since the edit was just added
	h.cur--
	h.end--
	*h.qmod(h.end) = nil
}

type EditHistoryEdit struct {
	ostr, str string
	strEdit   *StrEdit
}

func NewEditHistoryEdit(str string) *EditHistoryEdit {
	return &EditHistoryEdit{ostr: str, str: str, strEdit: &StrEdit{}}
}
func (he *EditHistoryEdit) Str() string {
	return he.str
}
func (he *EditHistoryEdit) Insert(index int, istr string) {
	he.str = he.strEdit.Insert(he.str, index, istr)
}
func (he *EditHistoryEdit) Delete(index, index2 int) {
	he.str = he.strEdit.Delete(he.str, index, index2)
}
func (he *EditHistoryEdit) Close() (string, *StrEdit, bool) {
	changed := he.str != he.ostr
	if !changed {
		return "", nil, false
	}
	return he.str, he.strEdit, true
}
