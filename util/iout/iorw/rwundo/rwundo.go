package rwundo

import (
	"github.com/jmigpin/editor/v2/util/iout/iorw"
	"github.com/jmigpin/editor/v2/util/iout/iorw/rwedit"
)

type RWUndo struct {
	iorw.ReadWriterAt
	History *History
}

func NewRWUndo(rw iorw.ReadWriterAt, hist *History) *RWUndo {
	rwu := &RWUndo{ReadWriterAt: rw, History: hist}
	return rwu
}

//----------

func (rw *RWUndo) OverwriteAt(i, n int, p []byte) error {
	// don't add to history if the result is equal
	changed := true
	if eq, err := iorw.REqual(rw, i, n, p); err == nil && eq {
		changed = false
	}

	ur, err := NewUndoRedoOverwrite(rw.ReadWriterAt, i, n, p)
	if err != nil {
		return err
	}

	if changed {
		edits := &Edits{}
		edits.Append(ur)
		rw.History.Append(edits)
	}
	return nil
}

//----------

func (rw *RWUndo) UndoRedo(redo, peek bool) (rwedit.SimpleCursor, bool, error) {
	edits, ok := rw.History.UndoRedo(redo, peek)
	if !ok {
		return rwedit.SimpleCursor{}, false, nil
	}
	c, err := edits.WriteUndoRedo(redo, rw.ReadWriterAt)
	if err != nil {
		// TODO: restore the undo/redo since it was not successful?
		return rwedit.SimpleCursor{}, false, err
	}
	return c, true, nil
}

//----------

// used in tests
func (rw *RWUndo) undo() (rwedit.SimpleCursor, bool, error) {
	return rw.UndoRedo(false, false)
}
func (rw *RWUndo) redo() (rwedit.SimpleCursor, bool, error) {
	return rw.UndoRedo(true, false)
}
