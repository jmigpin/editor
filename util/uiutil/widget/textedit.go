package widget

import (
	"image"

	"github.com/jmigpin/editor/util/evreg"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/iout/iorw/rwedit"
	"github.com/jmigpin/editor/util/iout/iorw/rwundo"
	"github.com/jmigpin/editor/util/uiutil/event"
)

//godebug:annotatefile

type TextEdit struct {
	*Text
	uiCtx   UIContext
	rwev    *iorw.RWEvents
	rwu     *rwundo.RWUndo
	ctx     *rwedit.Ctx     // ctx for rw editing utils (contains cursor)
	RWEvReg *evreg.Register // the rwundo wraps the rwev, so on a write event callback, the undo data is not commited yet. It is incorrect to try to undo inside a write callback. If a rwev wraps rwundo, undoing will not trigger the outer rwev events, otherwise undoing would register as another undo event (cycle).
}

func NewTextEdit(uiCtx UIContext) *TextEdit {
	t := NewText(uiCtx)
	te := &TextEdit{Text: t, uiCtx: uiCtx}

	te.rwev = iorw.NewRWEvents(te.Text.rw)
	te.RWEvReg = &te.rwev.EvReg
	te.RWEvReg.Add(iorw.RWEvIdWrite2, te.onWrite2)

	hist := rwundo.NewHistory(200)
	te.rwu = rwundo.NewRWUndo(te.rwev, hist)

	te.ctx = rwedit.NewCtx()
	te.ctx.RW = te.rwu
	te.ctx.C = rwedit.NewTriggerCursor(te.onCursorChange)
	te.ctx.Fns.Error = uiCtx.Error
	te.ctx.Fns.GetPoint = te.GetPoint
	te.ctx.Fns.GetIndex = te.GetIndex
	te.ctx.Fns.LineHeight = te.LineHeight
	te.ctx.Fns.MakeIndexVisible = te.MakeIndexVisible
	te.ctx.Fns.Undo = te.Undo
	te.ctx.Fns.Redo = te.Redo
	te.ctx.Fns.SetClipboardData = te.uiCtx.SetClipboardData
	te.ctx.Fns.GetClipboardData = func(i event.ClipboardIndex, fn func(string, error)) {
		te.uiCtx.GetClipboardData(i, func(s string, err error) {
			te.uiCtx.RunOnUIGoRoutine(func() {
				fn(s, err)
			})
		})
	}

	return te
}

//----------

func (te *TextEdit) RW() iorw.ReadWriterAt {
	// TODO: returning rw with undo/events, differs from SetRW(), workaround is to use te.Text.RW() to get underlying rw

	return te.ctx.RW
}

func (te *TextEdit) SetRW(rw iorw.ReadWriterAt) {
	// TODO: setting basic rw (bytes), differs from RW()

	te.Text.SetRW(rw)
	te.rwev.ReadWriterAt = rw
}

func (te *TextEdit) SetRWFromMaster(m *TextEdit) {
	te.SetRW(m.Text.rw)
	te.rwu.History = m.rwu.History
}

//----------

// Called when the changes are done on this textedit
func (te *TextEdit) onWrite2(ev interface{}) {
	e := ev.(*iorw.RWEvWrite2)
	if e.Changed {
		te.contentChanged()
	}
}

// Called when changes were made on another row
func (te *TextEdit) HandleRWWrite2(ev *iorw.RWEvWrite2) {
	te.stableRuneOffset(&ev.RWEvWrite)
	te.stableCursor(&ev.RWEvWrite)
	if ev.Changed {
		te.contentChanged()
	}
}

//----------

func (te *TextEdit) EditCtx() *rwedit.Ctx {
	return te.ctx
}

//----------

func (te *TextEdit) onCursorChange() {
	te.Drawer.SetCursorOffset(te.CursorIndex())
	te.MarkNeedsPaint()
}

//----------

func (te *TextEdit) Cursor() rwedit.Cursor {
	return te.ctx.C
}

func (te *TextEdit) CursorIndex() int {
	return te.Cursor().Index()
}

func (te *TextEdit) SetCursorIndex(i int) {
	te.Cursor().SetIndex(i)
}

//----------

func (te *TextEdit) Undo() error { return te.undoRedo(false) }
func (te *TextEdit) Redo() error { return te.undoRedo(true) }
func (te *TextEdit) undoRedo(redo bool) error {
	c, ok, err := te.rwu.UndoRedo(redo, false)
	if err != nil {
		return err
	}
	if ok {
		te.ctx.C.Set(c) // restore cursor
		te.MakeCursorVisible()
	}
	return nil
}

func (te *TextEdit) ClearUndones() {
	te.rwu.History.ClearUndones()
}

//----------

func (te *TextEdit) BeginUndoGroup() {
	c := te.ctx.C.Get()
	te.rwu.History.BeginUndoGroup(c)
}

func (te *TextEdit) EndUndoGroup() {
	c := te.ctx.C.Get()
	te.rwu.History.EndUndoGroup(c)
}

//----------

func (te *TextEdit) OnInputEvent(ev interface{}, p image.Point) event.Handled {
	te.BeginUndoGroup()
	defer te.EndUndoGroup()

	handled, err := rwedit.HandleInput(te.ctx, ev)
	if err != nil {
		te.uiCtx.Error(err)
	}
	return handled
}

//----------

func (te *TextEdit) SetBytes(b []byte) error {
	te.BeginUndoGroup()
	defer te.EndUndoGroup()
	defer func() {
		// because after setbytes the possible selection might not be correct (ex: go fmt; variable renames with lsprotorename)
		te.ctx.C.SetSelectionOff()
	}()
	return iorw.SetBytes(te.ctx.RW, b)
}

func (te *TextEdit) SetBytesClearPos(b []byte) error {
	te.BeginUndoGroup()
	defer te.EndUndoGroup()
	err := iorw.SetBytes(te.ctx.RW, b)
	te.ClearPos() // keep position in undogroup (history record)
	return err
}

// Keeps position (useful for file save)
func (te *TextEdit) SetBytesClearHistory(b []byte) error {
	te.rwu.History.Clear()
	rw := te.rwu.ReadWriterAt // bypass history
	if err := iorw.SetBytes(rw, b); err != nil {
		return err
	}
	return nil
}

func (te *TextEdit) AppendBytesClearHistory(b []byte) error {
	te.rwu.History.Clear()
	rw := te.rwu.ReadWriterAt // bypass history
	if err := rw.OverwriteAt(rw.Max(), 0, b); err != nil {
		return err
	}
	return nil
}

//----------

func (te *TextEdit) SetStr(str string) error {
	return te.SetBytes([]byte(str))
}

func (te *TextEdit) SetStrClearPos(str string) error {
	return te.SetBytesClearPos([]byte(str))
}

func (te *TextEdit) SetStrClearHistory(str string) error {
	return te.SetBytesClearHistory([]byte(str))
}

//----------

func (te *TextEdit) ClearPos() {
	te.ctx.C.SetIndexSelectionOff(0)
	te.MakeIndexVisible(0)
}

//----------

func (te *TextEdit) MakeCursorVisible() {
	if a, b, ok := te.ctx.C.SelectionIndexes(); ok {
		te.MakeRangeVisible(a, b-a)
	} else {
		te.MakeIndexVisible(te.ctx.C.Index())
	}
}

//----------

func (te *TextEdit) stableRuneOffset(ev *iorw.RWEvWrite) {
	// keep offset based scrolling stable
	ro := StableOffsetScroll(te.RuneOffset(), ev.Index, ev.Dn, ev.In)
	te.SetRuneOffset(ro)
}

func (te *TextEdit) stableCursor(ev *iorw.RWEvWrite) {
	c := te.Cursor()
	ci := StableOffsetScroll(c.Index(), ev.Index, ev.Dn, ev.In)
	if c.HaveSelection() {
		si := StableOffsetScroll(c.SelectionIndex(), ev.Index, ev.Dn, ev.In)
		c.SetSelection(si, ci)
	} else {
		te.SetCursorIndex(ci)
	}
}
