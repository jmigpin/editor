package iorw

type UndoRedo struct {
	Insert bool // otherwise is a delete
	Index  int
	B      []byte
}

func (ur *UndoRedo) Apply(w Writer, redo bool) error {
	if (ur.Insert && !redo) || (!ur.Insert && redo) {
		return w.Insert(ur.Index, ur.B)
	} else {
		return w.Delete(ur.Index, len(ur.B))
	}
}

//----------

func InsertUndoRedo(w Writer, i int, p []byte) (*UndoRedo, error) {
	if err := w.Insert(i, p); err != nil {
		return nil, err
	}
	b := make([]byte, len(p))
	copy(b, p)
	ur := &UndoRedo{Insert: false, Index: i, B: b}
	return ur, nil
}

func DeleteUndoRedo(rw ReadWriter, i, len int) (*UndoRedo, error) {
	b, err := rw.ReadNCopyAt(i, len)
	if err != nil {
		return nil, err
	}

	if err := rw.Delete(i, len); err != nil {
		return nil, err
	}

	ur := &UndoRedo{Insert: true, Index: i, B: b}
	return ur, nil
}

func OverwriteUndoRedo(rw ReadWriter, i, length int, p []byte) (_, _ *UndoRedo, _ error) {
	// copy delete
	b1, err := rw.ReadNCopyAt(i, length)
	if err != nil {
		return nil, nil, err
	}
	// copy insert
	b2 := make([]byte, len(p))
	copy(b2, p)
	// overwrite
	if err := rw.Overwrite(i, length, p); err != nil {
		return nil, nil, err
	}
	// delete/insert undoredo
	ur1 := &UndoRedo{Insert: true, Index: i, B: b1}
	ur2 := &UndoRedo{Insert: false, Index: i, B: b2}
	return ur1, ur2, nil
}
