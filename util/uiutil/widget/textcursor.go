package widget

import (
	"github.com/jmigpin/editor/util/drawutil/drawer4"
	"github.com/jmigpin/editor/util/iout/iorw"
)

type TextCursor struct {
	te      *TextEdit
	state   TextCursorState
	editing bool
	tcrw    *tcRW
}

func NewTextCursor(te *TextEdit) *TextCursor {
	tc := &TextCursor{te: te}
	tc.tcrw = &tcRW{ReadWriter: te.brw, tc: tc}
	return tc
}

//------------

func (tc *TextCursor) RW() iorw.ReadWriter {
	return tc.tcrw
}

//----------

func (tc *TextCursor) Edit(fn func()) {
	tc.BeginEdit()
	defer tc.EndEdit()
	fn()
}

func (tc *TextCursor) BeginEdit() {
	tc.panicIfEditing()

	tc.editing = true
	tc.te.TextHistory.BeginEdit()
}

func (tc *TextCursor) EndEdit() {
	tc.panicIfNotEditing()

	defer tc.te.contentChanged()

	tc.te.TextHistory.EndEdit()
	tc.editing = false
}

//----------

func (tc *TextCursor) panicIfNotEditing() {
	if !tc.editing {
		panic("edit mode is not set")
	}
}

func (tc *TextCursor) panicIfEditing() {
	if tc.editing {
		panic("edit mode is set")
	}
}

//----------

func (tc *TextCursor) Index() int {
	return tc.state.index
}

func (tc *TextCursor) SetIndex(index int) {
	if tc.state.index != index {
		tc.state.index = index

		if d, ok := tc.te.Drawer.(*drawer4.Drawer); ok {
			d.SetCursorOffset(tc.state.index)
		}

		tc.te.MarkNeedsPaint()
	}
}

func (tc *TextCursor) SelectionOn() bool {
	return tc.state.selectionOn
}

func (tc *TextCursor) SetSelectionOff() {
	if tc.state.selectionOn != false {
		tc.state.selectionIndex = 0
		tc.state.selectionOn = false
		tc.te.MarkNeedsPaint()
	}
}

func (tc *TextCursor) SelectionIndex() int {
	return tc.state.selectionIndex
}

func (tc *TextCursor) SetSelection(si, ci int) {
	tc.SetIndex(ci)
	if ci == si {
		tc.SetSelectionOff()
	} else {
		tc.state.selectionOn = true
		if tc.state.selectionIndex != si {
			tc.state.selectionIndex = si
			tc.te.MarkNeedsPaint()
		}
	}
}

func (tc *TextCursor) SetSelectionUpdate(on bool, ci int) {
	if on {
		si := tc.Index()
		if tc.SelectionOn() {
			si = tc.SelectionIndex()
		}
		tc.SetSelection(si, ci)
	} else {
		tc.SetSelectionOff()
		tc.SetIndex(ci)
	}
}

//----------

func (tc *TextCursor) SelectionIndexes() (int, int) {
	if !tc.SelectionOn() {
		panic("selection is not on")
	}
	a := tc.SelectionIndex()
	b := tc.Index()
	if a > b {
		a, b = b, a
	}
	return a, b
}

func (tc *TextCursor) Selection() ([]byte, error) {
	a, b := tc.SelectionIndexes()
	return tc.RW().ReadNCopyAt(a, b-a)
}

//----------

func (tc *TextCursor) LinesIndexes() (int, int, bool, error) {
	var a, b int
	if tc.SelectionOn() {
		a, b = tc.SelectionIndexes()
	} else {
		a = tc.Index()
		b = a
	}
	return iorw.LinesIndexes(tc.RW(), a, b)
}

//----------

type TextCursorState struct {
	index          int
	selectionOn    bool
	selectionIndex int
}

//----------

// Keeps history UndoRedo on write operations.
type tcRW struct {
	iorw.ReadWriter
	tc *TextCursor
}

func (rw *tcRW) Insert(i int, p []byte) error {
	rw.tc.panicIfNotEditing()

	ur, err := iorw.InsertUndoRedo(rw.ReadWriter, i, p)
	if err != nil {
		return err
	}
	rw.tc.te.TextHistory.Append(ur)
	return nil
}

func (rw *tcRW) Delete(i, len int) error {
	rw.tc.panicIfNotEditing()

	ur, err := iorw.DeleteUndoRedo(rw.ReadWriter, i, len)
	if err != nil {
		return err
	}
	rw.tc.te.TextHistory.Append(ur)
	return nil
}
