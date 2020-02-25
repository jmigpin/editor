package iorw

type UndoRedo struct {
	Type  WriterOp
	Index int
	B     []byte // insert/delete
	B2    []byte // overwrite insert
}

func (ur *UndoRedo) Apply(w Writer, redo bool) error {
	switch ur.Type {
	case WopInsert, WopDelete:
		insert := ur.Type == WopInsert
		if (insert && !redo) || (!insert && redo) {
			return w.Insert(ur.Index, ur.B)
		} else {
			return w.Delete(ur.Index, len(ur.B))
		}
	case WopOverwrite:
		if !redo {
			return w.Overwrite(ur.Index, len(ur.B), ur.B2)
		} else {
			return w.Overwrite(ur.Index, len(ur.B2), ur.B)
		}
	}
	panic("unexpected op")
}

//----------

func InsertUndoRedo(w Writer, i int, p []byte) (*UndoRedo, error) {
	if err := w.Insert(i, p); err != nil {
		return nil, err
	}
	b := make([]byte, len(p))
	copy(b, p)
	ur := &UndoRedo{Type: WopDelete, Index: i, B: b}
	return ur, nil
}

func DeleteUndoRedo(rw ReadWriter, i, n int) (*UndoRedo, error) {
	b, err := rw.ReadNCopyAt(i, n)
	if err != nil {
		return nil, err
	}

	if err := rw.Delete(i, n); err != nil {
		return nil, err
	}

	ur := &UndoRedo{Type: WopInsert, Index: i, B: b}
	return ur, nil
}

func OverwriteUndoRedo(rw ReadWriter, i, length int, p []byte) (*UndoRedo, error) {
	// copy delete
	b1, err := rw.ReadNCopyAt(i, length)
	if err != nil {
		return nil, err
	}
	// copy insert
	b2 := make([]byte, len(p))
	copy(b2, p)
	// overwrite
	if err := rw.Overwrite(i, length, p); err != nil {
		return nil, err
	}
	// delete/insert undoredo
	ur := &UndoRedo{Type: WopOverwrite, Index: i, B: b2, B2: b1}
	return ur, nil
}

//----------

type WriterOp int

const (
	WopInsert WriterOp = iota
	WopDelete
	WopOverwrite
)
