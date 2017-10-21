package tahistory

import "container/list"

type Edit struct {
	list.List
}

func (ed *Edit) Insert(str string, index int, istr string) string {
	if len(istr) == 0 {
		return str
	}
	id := &InsertDelete{true, index, istr, 0, 0}
	ed.PushBack(id)
	s, _ := id.apply(str, false)
	return s
}
func (ed *Edit) Delete(str string, index, index2 int) string {
	if index == index2 {
		return str
	}
	istr := str[index:index2]
	id := &InsertDelete{false, index, istr, 0, 0}
	ed.PushBack(id)
	s, _ := id.apply(str, false)
	return s
}

func (ed *Edit) Apply(str string) (string, int) {
	return ed.apply2(str, false)
}
func (ed *Edit) ApplyUndo(str string) (string, int) {
	return ed.apply2(str, true)
}
func (ed *Edit) apply2(str string, undo bool) (string, int) {
	c := 0
	if undo {
		for e := ed.Back(); e != nil; e = e.Prev() {
			id := e.Value.(*InsertDelete)
			str, c = id.apply(str, undo)
		}
	} else {
		for e := ed.Front(); e != nil; e = e.Next() {
			id := e.Value.(*InsertDelete)
			str, c = id.apply(str, undo)
		}
	}
	return str, c
}

func (ed *Edit) SetOpenCloseCursors(c1, c2 int) {
	if ed.Len() > 0 {
		ed.Front().Value.(*InsertDelete).cursorBefore = c1
		ed.Back().Value.(*InsertDelete).cursorAfter = c2
	}
}

type InsertDelete struct {
	IsInsert                  bool
	index                     int
	str                       string
	cursorBefore, cursorAfter int
}

func (id *InsertDelete) IsDel() bool {
	return !id.IsInsert
}

func (id *InsertDelete) apply(str string, undo bool) (string, int) {
	var str2 string
	if (id.IsInsert && !undo) || (!id.IsInsert && undo) {
		str2 = str[:id.index] + id.str + str[id.index:]
	} else {
		str2 = str[:id.index] + str[id.index+len(id.str):]
	}
	c := id.cursorAfter
	if undo {
		c = id.cursorBefore
	}
	return str2, c
}
