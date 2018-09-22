package history

import (
	"container/list"

	"github.com/jmigpin/editor/util/iout"
)

type Edit struct {
	list      list.List
	PreState  interface{}
	PostState interface{}
}

func (edit *Edit) Append(data *iout.UndoRedo) {
	edit.list.PushBack(data)
}

func (edit *Edit) Entries() []*iout.UndoRedo {
	w := make([]*iout.UndoRedo, edit.list.Len())
	i := 0
	for e := edit.list.Front(); e != nil; e = e.Next() {
		ur := e.Value.(*iout.UndoRedo)
		w[i] = ur
		i++
	}
	return w
}

func (edit *Edit) Empty() bool {
	for e := edit.list.Front(); e != nil; e = e.Next() {
		ur := e.Value.(*iout.UndoRedo)
		if len(ur.S) > 0 {
			return false
		}
	}
	return true
}

func (edit *Edit) ApplyUndoRedo(w iout.Writer, redo bool, restore func(interface{})) error {
	if redo {
		for e := edit.list.Front(); e != nil; e = e.Next() {
			ur := e.Value.(*iout.UndoRedo)
			if err := ur.Apply(w, redo); err != nil {
				return err
			}
		}
		restore(edit.PostState)
	} else {
		for e := edit.list.Back(); e != nil; e = e.Prev() {
			ur := e.Value.(*iout.UndoRedo)
			if err := ur.Apply(w, redo); err != nil {
				return err
			}
		}
		restore(edit.PreState)
	}
	return nil
}
