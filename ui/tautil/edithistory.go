package tautil

type EditHistory struct {
	q               []*StrEdit // edits queue
	start, end, cur int        // q positions
}

func NewEditHistory(n int) *EditHistory {
	return &EditHistory{q: make([]*StrEdit, n)}
}
func (eh *EditHistory) PushEdit(edit *StrEdit) {
	eh.q[eh.cur%len(eh.q)] = edit
	eh.cur++
	eh.end = eh.cur
	if eh.end-eh.start > len(eh.q) {
		eh.start = eh.end - len(eh.q)
	}
}
func (eh *EditHistory) PopUndo(str string) (string, int, bool) {
	if eh.cur-1 < eh.start {
		return "", 0, false // no undos
	}
	eh.cur--
	edit := eh.q[eh.cur%len(eh.q)]
	s, i := edit.undos.Apply(str)
	return s, i, true
}
func (eh *EditHistory) UnpopRedo(str string) (string, int, bool) {
	if eh.cur == eh.end {
		return "", 0, false // no redos
	}
	edit := eh.q[eh.cur%len(eh.q)]
	eh.cur++
	s, i := edit.edits.Apply(str)
	return s, i, true
}
func (eh *EditHistory) ClearQ() {
	eh.start, eh.cur, eh.end = 0, 0, 0
	for i := range eh.q {
		eh.q[i] = nil
	}
}

type EditHistoryEdit struct {
	ostr, str string
	strEdit   *StrEdit
}

func NewEditHistoryEdit(str string) *EditHistoryEdit {
	return &EditHistoryEdit{ostr: str, str: str, strEdit: &StrEdit{}}
}
func (ehe *EditHistoryEdit) Str() string {
	return ehe.str
}
func (ehe *EditHistoryEdit) Insert(index int, istr string) {
	ehe.str = ehe.strEdit.Insert(ehe.str, index, istr)
}
func (ehe *EditHistoryEdit) Delete(index, index2 int) {
	ehe.str = ehe.strEdit.Delete(ehe.str, index, index2)
}
func (ehe *EditHistoryEdit) Close() (string, *StrEdit, bool) {
	changed := ehe.str != ehe.ostr
	if !changed {
		return "", nil, false
	}
	return ehe.str, ehe.strEdit, true
}
