package rwundo

import "github.com/jmigpin/editor/v2/util/iout/iorw"

type UndoRedo struct {
	Index int
	D     []byte // deleted bytes of the original op
	I     []byte // inserted bytes of the original op
}

func NewUndoRedoOverwrite(rw iorw.ReadWriterAt, i, n int, p []byte) (*UndoRedo, error) {
	// copy delete
	b0, err := rw.ReadFastAt(i, n)
	if err != nil {
		return nil, err
	}
	b1 := iorw.MakeBytesCopy(b0)
	// copy insert
	b2 := make([]byte, len(p))
	copy(b2, p)

	if err := rw.OverwriteAt(i, n, p); err != nil {
		return nil, err
	}
	ur := &UndoRedo{Index: i, D: b1, I: b2}
	return ur, nil
}

//----------

func (ur *UndoRedo) Apply(redo bool, w iorw.WriterAt) error {
	if redo {
		return w.OverwriteAt(ur.Index, len(ur.D), ur.I)
	} else {
		return w.OverwriteAt(ur.Index, len(ur.I), ur.D)
	}
}

func (ur *UndoRedo) IsInsertOnly() bool {
	return len(ur.D) == 0 && len(ur.I) != 0
}
func (ur *UndoRedo) IsDeleteOnly() bool {
	return len(ur.D) != 0 && len(ur.I) == 0
}
