package iorw

type UndoRedo struct {
	Insert bool // otherwise is a delete
	Index  int
	S      []byte
}

func (ur *UndoRedo) Apply(w Writer, redo bool) error {
	if (ur.Insert && !redo) || (!ur.Insert && redo) {
		return w.Insert(ur.Index, ur.S)
	} else {
		return w.Delete(ur.Index, len(ur.S))
	}
}

//----------

func InsertUndoRedo(w Writer, i int, p []byte) (*UndoRedo, error) {
	if err := w.Insert(i, p); err != nil {
		return nil, err
	}
	s := make([]byte, len(p))
	copy(s, p)
	ur := &UndoRedo{Insert: false, Index: i, S: s}
	return ur, nil
}

func DeleteUndoRedo(rw ReadWriter, i, len int) (*UndoRedo, error) {
	s, err := rw.ReadNCopyAt(i, len)
	if err != nil {
		return nil, err
	}

	if err := rw.Delete(i, len); err != nil {
		return nil, err
	}

	ur := &UndoRedo{Insert: true, Index: i, S: s}
	return ur, nil
}
