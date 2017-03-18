package tautil

import "unicode"

type EditHistory struct {
	q               []*StrEdit // edits queue
	start, end, cur int        // q positions, start/end are always positive
}

func NewEditHistory(n int) *EditHistory {
	return &EditHistory{q: make([]*StrEdit, n)}
}
func (h *EditHistory) PushEdit(edit *StrEdit) {
	h.q[h.cur%len(h.q)] = edit
	h.cur++
	h.end = h.cur
	if h.end-h.start > len(h.q) {
		h.start = h.end - len(h.q)
	}
}
func (h *EditHistory) PopUndo(str string) (string, int, bool) {
	if h.cur-1 < h.start {
		return "", 0, false // no undos
	}
	h.cur--
	edit := h.q[h.cur%len(h.q)]
	s, i := edit.undos.Apply(str)
	return s, i, true
}
func (h *EditHistory) UnpopRedo(str string) (string, int, bool) {
	if h.cur == h.end {
		return "", 0, false // no redos
	}
	edit := h.q[h.cur%len(h.q)]
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
func (h *EditHistory) TryToMergeLastTwoEdits() {
	e1 := h.cur - 2
	e2 := h.cur - 1
	if e1 < h.start || e2 < h.start {
		return
	}
	edit1 := h.q[e1]
	edit2 := h.q[e2]

	editIsLetter := func(edit *StrEdit) (int, bool) {
		l := len(edit.edits)
		sei, ok := edit.edits[l-1].(*StrEditInsert)
		if !ok {
			return 0, false
		}
		if len(sei.str) != 1 {
			return 0, false
		}
		if !unicode.IsLetter(rune(sei.str[0])) {
			return 0, false
		}
		return sei.index, true
	}

	// merge consecutive letters
	a1, ok := editIsLetter(edit1)
	if !ok {
		return
	}
	a2, ok := editIsLetter(edit2)
	if !ok {
		return
	}
	if a1 != a2-1 { // TODO: only supporting size 1
		return
	}

	// add edit2 edits to edit1
	edit1.edits = append(edit1.edits, edit2.edits...)
	edit1.undos = append(edit2.undos, edit1.undos...)

	// remove edit2 from q to be rewritten
	h.cur--
	h.end--
	h.q[h.end] = nil
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
