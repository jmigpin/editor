package tahistory

import "container/list"

type StrEdit struct {
	edits StrEditActions
	undos StrEditActions
}

func (se *StrEdit) Insert(str string, index int, istr string) string {
	if len(istr) == 0 {
		return str
	}
	ins := &StrEditInsert{index, istr}
	se.edits.PushBack(ins)
	se.undos.PushFront(ins.getUndo())
	return ins.apply(str)
}
func (se *StrEdit) Delete(str string, index, index2 int) string {
	if index == index2 {
		return str
	}
	del := &StrEditDelete{index, index2}
	se.edits.PushBack(del)
	se.undos.PushFront(del.getUndo(str))
	return del.apply(str)
}
func (se *StrEdit) WrapLastEditWithPosition(index int) {
	ind := &StrEditPosition{index: index}
	se.edits.PushBack(ind)
	if se.undos.Front() == nil {
		se.edits.PushFront(ind)
	} else {
		// wrap last edit, usually a full string that throws the position to the end
		se.undos.InsertAfter(ind, se.undos.Front())
	}
}

func (se *StrEdit) IsEmpty() bool {
	return se.edits.Len() == 0
}

type StrEditActions struct {
	list.List
}

func (l *StrEditActions) Apply(str string) (string, int) {
	i := 0
	for e := l.Front(); e != nil; e = e.Next() {
		switch t0 := e.Value.(type) {
		case *StrEditInsert:
			str = t0.apply(str)
			i = t0.index + len(t0.str)
		case *StrEditDelete:
			str = t0.apply(str)
			i = t0.index
		case *StrEditPosition:
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

type StrEditPosition struct {
	index int
}
