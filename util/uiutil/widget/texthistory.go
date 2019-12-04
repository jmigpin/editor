package widget

import (
	"image"
	"log"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget/history"
)

// Editable string history
type TextHistory struct {
	hist *history.History
	te   *TextEdit
	edit *history.Edit
}

func NewTextHistory(es *TextEdit) *TextHistory {
	th := &TextHistory{
		hist: history.NewHistory(200),
		te:   es,
	}
	return th
}

//----------

func (th *TextHistory) clear() {
	th.hist.Clear()
}
func (th *TextHistory) ClearForward() {
	th.hist.ClearForward()
}
func (th *TextHistory) New(maxEntries int) {
	th.hist = history.NewHistory(maxEntries)
}
func (th *TextHistory) Use(th2 *TextHistory) {
	th.hist = th2.hist
}

//----------

func (th *TextHistory) BeginEdit() {
	if th.edit != nil {
		panic("already editing")
	}
	th.edit = &history.Edit{}
	th.edit.PreState = th.cursorState()
}

func (th *TextHistory) EndEdit() {
	cleanup := func() {
		th.edit = nil
	}
	defer cleanup()

	th.edit.PostState = th.cursorState()

	th.hist.Append(th.edit)
}

//----------

func (th *TextHistory) Append(ur *iorw.UndoRedo) {
	th.edit.Append(ur)
}

//----------

func (th *TextHistory) cursorState() interface{} {
	return th.te.TextCursor.state
}

func (th *TextHistory) restoreCursorState(data interface{}) {
	state := data.(TextCursorState)

	// set state through the proper function calls (can't assign directly)
	tc := th.te.TextCursor
	if state.selectionOn {
		tc.SetSelection(state.selectionIndex, state.index)
	} else {
		tc.SetSelectionOff()
		tc.SetIndex(state.index)
	}

	// make index visible
	if !tc.SelectionOn() {
		tc.te.MakeIndexVisible(tc.Index())
	} else {
		a, b := tc.SelectionIndex(), tc.Index()
		if a > b {
			a, b = b, a
		}
		tc.te.MakeRangeVisible(a, b-a)
	}
}

//----------

func (th *TextHistory) Undo() error { return th.undoRedo(false) }
func (th *TextHistory) Redo() error { return th.undoRedo(true) }

func (th *TextHistory) undoRedo(redo bool) error {
	th.te.TextCursor.panicIfEditing()

	edit := th.hist.UndoRedo(redo)
	if edit == nil {
		return nil
	}

	defer th.te.contentChanged()

	restore := func(data interface{}) {
		th.restoreCursorState(data) // makes index visible (triggers paint)
	}
	return edit.ApplyUndoRedo(th.te.crw, redo, restore)
}

//----------

func (th *TextHistory) HandleInputEvent(ev0 interface{}, p image.Point) event.Handled {
	switch ev := ev0.(type) {
	case *event.KeyDown:
		switch {
		case ev.Mods.ClearLocks().Is(event.ModCtrl | event.ModShift):
			switch ev.KeySym {
			case event.KSymZ:
				// TODO: error context
				if err := th.Redo(); err != nil {
					log.Print(err)
				}
				return event.HTrue
			}
		case ev.Mods.ClearLocks().Is(event.ModCtrl):
			switch ev.KeySym {
			case event.KSymZ:
				// TODO: error context
				if err := th.Undo(); err != nil {
					log.Print(err)
				}
				return event.HTrue
			}
		}
	}
	return event.HFalse
}
