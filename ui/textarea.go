package ui

import (
	"image"
	"log"

	"github.com/jmigpin/editor/drawutil"
	"github.com/jmigpin/editor/imageutil"
	"github.com/jmigpin/editor/ui/tautil"
	"github.com/jmigpin/editor/uiutil"
	"github.com/jmigpin/editor/xutil/keybmap"
	"github.com/jmigpin/editor/xutil/xgbutil"

	"github.com/BurntSushi/xgb/xproto"

	"golang.org/x/image/math/fixed"
)

type TextArea struct {
	C             uiutil.Container
	ui            *UI
	buttonPressed bool
	EvReg         *xgbutil.EventRegister
	dereg         xgbutil.EventDeregister
	stringCache   *drawutil.StringCache

	Colors                     *drawutil.Colors
	DisableHighlightCursorWord bool
	DisableButtonScroll        bool

	str         string
	cursorIndex int
	offsetY     fixed.Int26_6
	selection   struct {
		on    bool
		index int // from index to cursorIndex
	}

	offsetIndexWidthChange int

	undo struct {
		edit            *TextAreaEdit   // current edit
		str             string          // str used while editing
		start, end, cur int             // positions
		q               []*TextAreaEdit // edits queue
	}

	//cache struct {
	//offsetIndex struct {
	//firstCalcDone bool
	//areaDx        int
	//}
	//}
}

func NewTextArea(ui *UI) *TextArea {
	ta := &TextArea{ui: ui}
	c := drawutil.DefaultColors()
	ta.Colors = &c
	ta.C.PaintFunc = ta.paint
	ta.undo.q = make([]*TextAreaEdit, 30)
	ta.stringCache = drawutil.NewStringCache(ui.FontFace())
	ta.EvReg = xgbutil.NewEventRegister()

	r1 := ta.ui.Win.EvReg.Add(keybmap.KeyPressEventId,
		&xgbutil.ERCallback{ta.onKeyPress})
	r2 := ta.ui.Win.EvReg.Add(keybmap.ButtonPressEventId,
		&xgbutil.ERCallback{ta.onButtonPress})
	r3 := ta.ui.Win.EvReg.Add(keybmap.ButtonReleaseEventId,
		&xgbutil.ERCallback{ta.onButtonRelease})
	r4 := ta.ui.Win.EvReg.Add(keybmap.MotionNotifyEventId,
		&xgbutil.ERCallback{ta.onMotionNotify})
	ta.dereg.Add(r1, r2, r3, r4)

	return ta
}
func (ta *TextArea) Close() {
	ta.dereg.UnregisterAll()
}

// Used externally for dynamic textarea height.
func (ta *TextArea) CalcStringHeight(width int) int {
	ta.stringCache.Update(ta.str, width)
	th := ta.stringCache.Height().Round()
	// minimum height (ex: empty text)
	min := ta.LineHeight().Round()
	if th < min {
		th = min
	}
	return th
}

func (ta *TextArea) updateStringCache() {
	ta.stringCache.Update(ta.str, ta.C.Bounds.Dx())
}

func (ta *TextArea) paint() {
	// fill background
	imageutil.FillRectangle(ta.ui.Image(), &ta.C.Bounds, ta.Colors.Bg)

	ta.updateStringCacheWithOffsetFix()

	selection := ta.getSelection()
	highlight := !ta.DisableHighlightCursorWord && selection == nil
	err := ta.stringCache.Draw(
		ta.ui.Image(),
		&ta.C.Bounds,
		ta.cursorIndex,
		ta.offsetY,
		ta.Colors,
		selection,
		highlight)
	if err != nil {
		ta.Error(err)
	}
}
func (ta *TextArea) getSelection() *drawutil.Selection {
	selectionVisible := ta.selection.index != ta.cursorIndex
	if ta.selection.on && selectionVisible {
		return &drawutil.Selection{
			StartIndex: ta.selection.index,
			EndIndex:   ta.cursorIndex,
		}
	}
	return nil
}
func (ta *TextArea) updateStringCacheWithOffsetFix() {
	fixOffset := false
	offsetIndex := 0
	if ta.C.Bounds.Dx() != ta.offsetIndexWidthChange {
		ta.offsetIndexWidthChange = ta.C.Bounds.Dx()
		fixOffset = true
		offsetIndex = ta.OffsetIndex()
		log.Printf("**ta offsetindex %v\n", offsetIndex)
	}

	ta.updateStringCache()

	if fixOffset {
		ta.SetOffsetIndex(offsetIndex)
	}
}

func (ta *TextArea) TextHeight() fixed.Int26_6 {
	return ta.stringCache.Height()
}

func (ta *TextArea) Error(err error) {
	ta.EvReg.Emit(TextAreaErrorEventId, err)
}

func (ta *TextArea) Str() string {
	if ta.undo.edit != nil {
		// return undo str while editing
		return ta.undo.str
	}
	return ta.str
}
func (ta *TextArea) setStr(s string) {
	if s == ta.str {
		return
	}
	ta.str = s
	ta.SetCursorIndex(ta.CursorIndex()) // ensure valid cursor
	oldBounds := ta.C.Bounds
	ta.updateStringCache()
	ta.C.NeedPaint()

	ev := &TextAreaSetTextEvent{ta, oldBounds}
	ta.EvReg.Emit(TextAreaSetTextEventId, ev)
}
func (ta *TextArea) SetStrClear(str string, clearPosition, clearUndoQ bool) {
	ta.SetSelectionOn(false)
	if clearPosition {
		ta.SetCursorIndex(0)
		ta.SetOffsetY(0)
	}
	if clearUndoQ {
		ta.clearUndoQ()
		ta.setStr(str)
	} else {
		ta.EditRemove(0, len(ta.str))
		ta.EditInsert(0, str)
		ta.EditDone()
	}
}

func (ta *TextArea) ensureEdit() {
	if ta.undo.edit == nil {
		ta.undo.edit = &TextAreaEdit{}
		// using a separate str instance to edit allows to detect if the edit actually changed the final string or not when calling for setStr()
		ta.undo.str = ta.str
	}
}
func (ta *TextArea) EditInsert(index int, str string) {
	ta.ensureEdit()
	ta.undo.str = ta.undo.edit.insert(ta.undo.str, index, str)
}
func (ta *TextArea) EditRemove(index, index2 int) {
	ta.ensureEdit()
	ta.undo.str = ta.undo.edit.remove(ta.undo.str, index, index2)
}
func (ta *TextArea) EditDone() {
	if ta.undo.edit == nil {
		panic("missing edit instance")
	}
	if !ta.undo.edit.IsEmpty() {
		ta.pushEdit(ta.undo.edit)
		ta.setStr(ta.undo.str)
	}
	ta.undo.edit = nil
	ta.undo.str = ""
}

func (ta *TextArea) pushEdit(edit *TextAreaEdit) {
	u := &ta.undo
	u.q[u.cur%len(u.q)] = edit
	u.cur++
	u.end = u.cur
	if u.end-u.start > len(u.q) {
		u.start = u.end - len(u.q)
	}
}
func (ta *TextArea) popUndo() {
	u := &ta.undo
	if u.cur-1 < u.start {
		return // no undos
	}
	u.cur--
	edit := u.q[u.cur%len(u.q)]
	s, i := edit.undos.apply(ta.str)
	ta.setStr(s)
	ta.SetCursorIndex(i)
	ta.SetSelectionOn(false)
}
func (ta *TextArea) unpopRedo() {
	u := &ta.undo
	if u.cur == u.end {
		return // no redos
	}
	edit := u.q[u.cur%len(u.q)]
	u.cur++
	s, i := edit.edits.apply(ta.str)
	ta.setStr(s)
	ta.SetCursorIndex(i)
	ta.SetSelectionOn(false)
}
func (ta *TextArea) clearUndoQ() {
	u := &ta.undo
	u.start, u.cur, u.end = 0, 0, 0
	for i := range u.q {
		u.q[i] = nil
	}
}

func (ta *TextArea) CursorIndex() int {
	return ta.cursorIndex
}
func (ta *TextArea) SetCursorIndex(v int) {
	if v < 0 {
		v = 0
	}
	if v > len(ta.str) {
		v = len(ta.str)
	}
	if v != ta.cursorIndex {
		ta.cursorIndex = v
		ta.C.NeedPaint()
	}
}

func (ta *TextArea) SelectionOn() bool {
	return ta.selection.on
}
func (ta *TextArea) SetSelectionOn(v bool) {
	if v != ta.selection.on {
		ta.selection.on = v
		ta.C.NeedPaint()
	}
}

func (ta *TextArea) SelectionIndex() int {
	return ta.selection.index
}
func (ta *TextArea) SetSelectionIndex(v int) {
	if v != ta.selection.index {
		ta.selection.index = v
		if ta.SelectionOn() {
			ta.C.NeedPaint()
		}
	}
}

func (ta *TextArea) OffsetY() fixed.Int26_6 {
	return ta.offsetY
}
func (ta *TextArea) SetOffsetY(v fixed.Int26_6) {
	if v < 0 {
		v = 0
	}
	if v > ta.TextHeight() {
		v = ta.TextHeight()
	}
	if v != ta.offsetY {
		ta.offsetY = v
		ta.C.NeedPaint()

		ev := &TextAreaSetOffsetYEvent{ta}
		ta.EvReg.Emit(TextAreaSetOffsetYEventId, ev)
	}
}

func (ta *TextArea) OffsetIndex() int {
	p := fixed.Point26_6{0, ta.offsetY}
	return ta.stringCache.GetIndex(&p)
}
func (ta *TextArea) SetOffsetIndex(i int) {
	p := ta.stringCache.GetPoint(i)
	ta.SetOffsetY(p.Y)
}

func (ta *TextArea) MakeIndexVisible(index int) {
	p := ta.stringCache.GetPoint(index)
	half := fixed.I(ta.C.Bounds.Dy() / 2)
	offsetY := p.Y - half
	ta.SetOffsetY(offsetY)
}
func (ta *TextArea) MakeCursorVisibleAndWarpPointerToCursor() {
	ta.MakeIndexVisible(ta.CursorIndex())

	p := ta.stringCache.GetPoint(ta.CursorIndex())
	p.Y -= ta.offsetY
	p2 := drawutil.Point266ToPoint(p)
	p3 := p2.Add(ta.C.Bounds.Min)
	// add pad
	p3.Y += ta.LineHeight().Round()
	p3.X += 5

	// ensure the cursor is reachable in X (ex: textarea is small and cursor is drawn outside of it)
	if !p3.In(ta.C.Bounds) {
		p3.X = 0
	}

	ta.ui.WarpPointer(&p3)
}

func (ta *TextArea) RequestTreePaint() {
	ta.ui.RequestTreePaint()
}
func (ta *TextArea) RequestClipboardString() (string, error) {
	return ta.ui.Win.Paste.Request()
}
func (ta *TextArea) SetClipboardString(v string) {
	ta.ui.Win.Copy.Set(v)
}
func (ta *TextArea) LineHeight() fixed.Int26_6 {
	fm := ta.ui.FontFace().Face.Metrics()
	return drawutil.LineHeight(&fm)
}
func (ta *TextArea) IndexPoint266(i int) *fixed.Point26_6 {
	return ta.stringCache.GetPoint(i)
}
func (ta *TextArea) Point266Index(p *fixed.Point26_6) int {
	return ta.stringCache.GetIndex(p)
}

// Drawn area point index.
func (ta *TextArea) PointIndexFromOffset(p *image.Point) int {
	p0i := p.Sub(ta.C.Bounds.Min)
	p0 := drawutil.PointToPoint266(&p0i)
	p0.Y += ta.offsetY
	return ta.stringCache.GetIndex(p0)
}

func (ta *TextArea) onButtonPress(ev0 xgbutil.EREvent) {
	ev := ev0.(*keybmap.ButtonPressEvent)
	if !ev.Point.In(ta.C.Bounds) {
		return
	}
	ta.buttonPressed = true
	switch ev.Button.Button {
	case xproto.ButtonIndex1:
		sel := ev.Button.Mods.Shift()
		tautil.MoveCursorToPoint(ta, ev.Point, sel)
	case xproto.ButtonIndex4:
		if !ta.DisableButtonScroll {
			tautil.ScrollUp(ta)
			ev2 := &TextAreaScrollEvent{ta, true}
			ta.EvReg.Emit(TextAreaScrollEventId, ev2)
		}
	case xproto.ButtonIndex5:
		if !ta.DisableButtonScroll {
			tautil.ScrollDown(ta)
			ev2 := &TextAreaScrollEvent{ta, false}
			ta.EvReg.Emit(TextAreaScrollEventId, ev2)
		}
	}
}
func (ta *TextArea) onButtonRelease(ev0 xgbutil.EREvent) {
	if !ta.buttonPressed {
		return
	}
	ta.buttonPressed = false
	ev := ev0.(*keybmap.ButtonReleaseEvent)
	switch ev.Button.Button {
	case xproto.ButtonIndex3: // 2=middle, 3=right
		// release must be in the area to run the cmd
		if ev.Point.In(ta.C.Bounds) {
			tautil.MoveCursorToPoint(ta, ev.Point, false)
			ev2 := &TextAreaCmdEvent{ta}
			ta.EvReg.Emit(TextAreaCmdEventId, ev2)
		}
	}
}
func (ta *TextArea) onMotionNotify(ev0 xgbutil.EREvent) {
	if !ta.buttonPressed {
		return
	}
	ta.ui.RequestMotionNotify()
	ev := ev0.(*keybmap.MotionNotifyEvent)
	if ev.Modifiers.Button1() {
		tautil.MoveCursorToPoint(ta, ev.Point, true)
	}
}
func (ta *TextArea) onKeyPress(ev0 xgbutil.EREvent) {
	ev := ev0.(*keybmap.KeyPressEvent)
	if !ev.Point.In(ta.C.Bounds) {
		return
	}
	k := ev.Key
	firstKeysym := k.FirstKeysym()
	switch firstKeysym {
	case keybmap.XKRight:
		sel := k.Modifiers.Shift()
		if k.Modifiers.Control() {
			tautil.MoveCursorJumpRight(ta, sel)
		} else {
			tautil.MoveCursorRight(ta, sel)
		}
	case keybmap.XKLeft:
		sel := k.Modifiers.Shift()
		if k.Modifiers.Control() {
			tautil.MoveCursorJumpLeft(ta, sel)
		} else {
			tautil.MoveCursorLeft(ta, sel)
		}
	case keybmap.XKUp:
		if k.Modifiers.Control() && k.Modifiers.Mod1() {
			tautil.MoveLineUp(ta)
		} else {
			sel := k.Modifiers.Shift()
			tautil.MoveCursorUp(ta, sel)
		}
	case keybmap.XKDown:
		if k.Modifiers.Control() && k.Modifiers.Mod1() {
			if k.Modifiers.Shift() {
				tautil.DuplicateLines(ta)
			} else {
				tautil.MoveLineDown(ta)
			}
		} else {
			sel := k.Modifiers.Shift()
			tautil.MoveCursorDown(ta, sel)
		}
	case keybmap.XKBackspace:
		tautil.Backspace(ta)
	case keybmap.XKDelete:
		tautil.Delete(ta)
	case keybmap.XKHome:
		sel := k.Modifiers.Shift()
		if k.Modifiers.Control() {
			tautil.StartOfString(ta, sel)
		} else {
			tautil.StartOfLine(ta, sel)
		}
	case keybmap.XKEnd:
		sel := k.Modifiers.Shift()
		if k.Modifiers.Control() {
			tautil.EndOfString(ta, sel)
		} else {
			tautil.EndOfLine(ta, sel)
		}
	default:
		// shortcuts with printable runes
		if k.Modifiers.Control() {
			switch firstKeysym {
			case 'd':
				if k.Modifiers.Shift() {
					tautil.Uncomment(ta)
				} else {
					tautil.Comment(ta)
				}
				return
			case 'c':
				tautil.Copy(ta)
				return
			case 'x':
				tautil.Cut(ta)
				return
			case 'v':
				tautil.Paste(ta)
				return
			case 'k':
				tautil.RemoveLines(ta)
				return
			case 'a':
				tautil.SelectAll(ta)
				return
			case 'z':
				if k.Modifiers.Shift() {
					ta.unpopRedo()
				} else {
					ta.popUndo()
				}
			}
		}
		switch firstKeysym {
		case keybmap.XKTab:
			if k.Modifiers.Shift() {
				tautil.TabLeft(ta)
				return
			}
			if ta.SelectionOn() {
				tautil.TabRight(ta)
				return
			}
		}

		ta.insertRuneInText(k)
	}
}
func (ta *TextArea) insertRuneInText(k *keybmap.Key) {
	// special runes checked from first keysym from keysym table
	switch k.FirstKeysym() {
	case keybmap.XKAltL,
		keybmap.XKIsoLevel3Shift,
		keybmap.XKShiftL,
		keybmap.XKShiftR,
		keybmap.XKControlL,
		keybmap.XKControlR,
		keybmap.XKPageUp,
		keybmap.XKPageDown,
		keybmap.XKCapsLock,
		keybmap.XKNumLock,
		keybmap.XKSuperL:
		// ignore these
		return
	case keybmap.XKReturn:
		tautil.InsertRune(ta, '\n')
	case keybmap.XKTab:
		tautil.InsertRune(ta, '\t')
	case keybmap.XKSpace:
		tautil.InsertRune(ta, ' ')
	default:
		// print rune from keysym table
		ks := k.Keysym()
		switch ks {
		case keybmap.XKAsciiTilde:
			tautil.InsertRune(ta, '~')
		case keybmap.XKAsciiCircum:
			tautil.InsertRune(ta, '^')
		case keybmap.XKAcute:
			tautil.InsertRune(ta, '´')
		case keybmap.XKGrave:
			tautil.InsertRune(ta, '`')
		default:
			// don't print if control is pressed
			if k.Modifiers.Control() {
				return
			}

			tautil.InsertRune(ta, rune(ks))

			// prevent stringcache calcrunedata
			//ta.stringCache.str = ta.Text()
		}
	}
}

const (
	TextAreaCmdEventId = iota
	TextAreaScrollEventId
	TextAreaSetTextEventId
	TextAreaSetOffsetYEventId
	TextAreaErrorEventId
)

type TextAreaCmdEvent struct {
	TextArea *TextArea
}
type TextAreaScrollEvent struct {
	TextArea *TextArea
	Up       bool
}
type TextAreaSetTextEvent struct {
	TextArea  *TextArea
	OldBounds image.Rectangle
}
type TextAreaSetOffsetYEvent struct {
	TextArea *TextArea
}
