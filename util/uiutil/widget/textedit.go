package widget

import (
	"image"
	"strings"

	"github.com/jmigpin/editor/util/evreg"
	"github.com/jmigpin/editor/util/fontutil"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/iout/iorw/rwedit"
	"github.com/jmigpin/editor/util/iout/iorw/rwundo"
	"github.com/jmigpin/editor/util/parseutil"
	"github.com/jmigpin/editor/util/uiutil/event"
)

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
	te.ctx.Fns.SetClipboardData = func(i event.ClipboardIndex, s string) {
		te.uiCtx.SetClipboardData(i, textEditClipboardString(s))
	}
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

	// TODO: should be set at instanciation when known that it will be a duplicate
	te.rwu.History = m.rwu.History
}

//----------

// Called when the changes are done on this textedit
func (te *TextEdit) onWrite2(ev any) {
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

func (te *TextEdit) OnInputEvent(ev any, p image.Point) event.Handled {
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
	ci := te.Cursor().Index()
	hasCursor := false
	atEnd := false
	var pos StableCursorPos
	if ci >= 0 && ci <= te.RW().Max() {
		hasCursor = true
		if ci == te.RW().Max() {
			atEnd = true
		} else {
			pos = GetStableCursorPos(te.RW(), ci)
		}
	}

	te.BeginUndoGroup()
	defer te.EndUndoGroup()
	defer func() {
		// because after setbytes the possible selection might not be correct (ex: go fmt; variable renames with lsprotorename)
		te.ctx.C.SetSelectionOff()
		if hasCursor {
			newIdx := 0
			if atEnd {
				newIdx = te.RW().Max()
			} else {
				newIdx = FindStableCursorIndex(te.RW(), pos)
			}
			te.ctx.C.SetIndex(newIdx)
		}
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
	return te.OverwriteBytesClearHistory(te.RW().Min(), te.RW().Max(), b)
}
func (te *TextEdit) AppendBytesClearHistory(b []byte) error {
	return te.OverwriteBytesClearHistory(te.RW().Max(), 0, b)
}
func (te *TextEdit) OverwriteBytesClearHistory(i, del int, b []byte) error {
	ci := te.Cursor().Index()
	hasCursor := false
	atEnd := false
	var pos StableCursorPos
	if ci >= 0 && ci <= te.RW().Max() {
		hasCursor = true
		if ci == te.RW().Max() {
			atEnd = true
		} else {
			pos = GetStableCursorPos(te.RW(), ci)
		}
	}

	te.rwu.History.Clear()
	rw := te.rwu.ReadWriterAt // bypass history
	if err := rw.OverwriteAt(i, del, b); err != nil {
		return err
	}

	if hasCursor {
		newIndex := 0
		if atEnd {
			newIndex = te.RW().Max()
		} else {
			newIndex = FindStableCursorIndex(te.RW(), pos)
		}
		te.Cursor().SetIndexSelectionOff(newIndex)
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

//----------
//----------
//----------

const stableCursorWindowSize = 2048

// StableCursorPos holds a relative cursor position from a window start, facilitating O(1) cursor visual position preservation.
type StableCursorPos struct {
	Offset int
	Line   int
	Col    int // rune column
}

// cursor position relative to a window start to avoid performance issues on large files.
func GetStableCursorPos(rw iorw.ReaderAt, ci int) StableCursorPos {
	if ci < 0 || ci > rw.Max() {
		return StableCursorPos{}
	}

	startOffset := ci - stableCursorWindowSize
	if startOffset < 0 {
		startOffset = 0
	}

	b, err := rw.ReadFastAt(startOffset, ci-startOffset)
	if err != nil {
		return StableCursorPos{Offset: startOffset}
	}

	line, col := parseutil.IndexLineColumnFn(b, len(b), isNlOrWrap)
	return StableCursorPos{
		Offset: startOffset,
		Line:   line,
		Col:    col,
	}
}

// maps a stable cursor position back into a byte index for the given text reader by scanning from the stored offset.
func FindStableCursorIndex(rw iorw.ReaderAt, pos StableCursorPos) int {
	offset := pos.Offset
	max := rw.Max()
	if offset > max {
		offset = max
	}

	chunkSize := stableCursorWindowSize*2 + pos.Line*100 + pos.Col*4
	if offset+chunkSize > max {
		chunkSize = max - offset
	}

	b, err := rw.ReadFastAt(offset, chunkSize)
	if err != nil {
		return offset
	}

	relOffset, err := parseutil.LineColumnIndexFn(b, pos.Line, pos.Col, isNlOrWrap)
	if err != nil {
		return rw.Max()
	}

	byteOffset := pos.Offset + relOffset
	if byteOffset > max {
		byteOffset = max
	}
	return byteOffset
}

//----------

func isNlOrWrap(ru rune) bool {
	return ru == '\n' || ru == fontutil.TermWrapContinuousRune
}

func textEditClipboardString(s string) string {
	return strings.ReplaceAll(s, string(rune(fontutil.TermWrapContinuousRune)), "")
}
