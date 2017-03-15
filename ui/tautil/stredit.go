package tautil

type StrEdit struct {
	edits StrEditActions
	undos StrEditActions
}

func (se *StrEdit) Insert(str string, index int, istr string) string {
	if len(istr) == 0 {
		return str
	}
	ins := &StrEditInsert{index, istr}
	se.edits = append(se.edits, ins)
	se.undos = append(StrEditActions{ins.getUndo()}, se.undos...)
	return ins.apply(str)
}
func (se *StrEdit) Delete(str string, index, index2 int) string {
	if index == index2 {
		return str
	}
	del := &StrEditDelete{index, index2}
	se.edits = append(se.edits, del)
	se.undos = append(StrEditActions{del.getUndo(str)}, se.undos...)
	return del.apply(str)
}
func (se *StrEdit) IsEmpty() bool {
	return len(se.edits) == 0
}

type StrEditActions []interface{} // inserts/deletes

func (u StrEditActions) Apply(str string) (string, int) {
	i := 0
	for _, e := range u {
		switch t0 := e.(type) {
		case *StrEditInsert:
			str = t0.apply(str)
			i = t0.index + len(t0.str)
		case *StrEditDelete:
			str = t0.apply(str)
			i = t0.index
		default:
			panic("!")
		}
	}
	return str, i
}

type StrEditInsert struct {
	index int
	str   string
}

func (u *StrEditInsert) apply(str string) string {
	return str[:u.index] + u.str + str[u.index:]
}
func (u *StrEditInsert) getUndo() *StrEditDelete {
	return &StrEditDelete{u.index, u.index + len(u.str)}
}

type StrEditDelete struct {
	index  int
	index2 int
}

func (u *StrEditDelete) apply(str string) string {
	return str[:u.index] + str[u.index2:]
}
func (u *StrEditDelete) getUndo(str string) *StrEditInsert {
	return &StrEditInsert{u.index, str[u.index:u.index2]}
}
