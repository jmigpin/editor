package rwundo

import (
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/iout/iorw/rwedit"
)

type RWUndo struct {
	iorw.ReadWriter
	History *History
}

func NewRWUndo(rw iorw.ReadWriter, hist *History) *RWUndo {
	rwu := &RWUndo{ReadWriter: rw, History: hist}
	return rwu
}

//----------

func (rw *RWUndo) Overwrite(i, n int, p []byte) error {
	ur, err := NewUndoRedoOverwrite(rw.ReadWriter, i, n, p)
	if err != nil {
		return err
	}

	// don't add to history if the result is equal
	if eq, err := iorw.REqual(rw, p); err == nil && eq {
		return nil
	}

	edits := &Edits{}
	edits.Append(ur)
	rw.History.Append(edits)
	return nil
}

//----------

func (rw *RWUndo) Undo() error { return rw.UndoRedo(false, nil) }
func (rw *RWUndo) Redo() error { return rw.UndoRedo(true, nil) }
func (rw *RWUndo) UndoRedo(redo bool, restore func(rwedit.CursorData)) error {
	edits, ok := rw.History.UndoRedo(redo)
	if !ok {
		return nil
	}
	if err := edits.WriteUndoRedo(redo, rw.ReadWriter, restore); err != nil {
		// TODO: restore the undo/redo since it was not successful?
		return err
	}
	return nil
}
