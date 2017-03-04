package ui

type TextAreaEdit struct {
	edits TextAreaEdits
	undos TextAreaEdits
}

func (tae *TextAreaEdit) insert(str string, index int, s string) string {
	if len(s) == 0 {
		return str
	}
	insert := &TextAreaEditInsert{index, s}
	tae.edits = append(tae.edits, insert)
	tae.undos = append(TextAreaEdits{insert.undo()}, tae.undos...)
	return insert.apply(str)
}
func (tae *TextAreaEdit) remove(str string, index, index2 int) string {
	if index == index2 {
		return str
	}
	remove := &TextAreaEditRemove{index, index2}
	tae.edits = append(tae.edits, remove)
	tae.undos = append(TextAreaEdits{remove.undo(str)}, tae.undos...)
	return remove.apply(str)
}
func (tae *TextAreaEdit) IsEmpty() bool {
	return len(tae.edits) == 0
}

type TextAreaEdits []interface{} // inserts/removes

func (u TextAreaEdits) apply(str string) (string, int) {
	i := 0
	for _, e := range u {
		switch t0 := e.(type) {
		case *TextAreaEditInsert:
			str = t0.apply(str)
			i = t0.index + len(t0.str)
		case *TextAreaEditRemove:
			str = t0.apply(str)
			i = t0.index
		default:
			panic("!")
		}
	}
	return str, i
}

type TextAreaEditInsert struct {
	index int
	str   string
}

func (u *TextAreaEditInsert) apply(str string) string {
	return str[:u.index] + u.str + str[u.index:]
}
func (u *TextAreaEditInsert) undo() *TextAreaEditRemove {
	return &TextAreaEditRemove{u.index, u.index + len(u.str)}
}

type TextAreaEditRemove struct {
	index  int
	index2 int
}

func (u *TextAreaEditRemove) apply(str string) string {
	return str[:u.index] + str[u.index2:]
}
func (u *TextAreaEditRemove) undo(str string) *TextAreaEditInsert {
	return &TextAreaEditInsert{u.index, str[u.index:u.index2]}
}
