package ui

import (
	"image"

	"github.com/jmigpin/editor/drawutil"
	"github.com/jmigpin/editor/imageutil"
	"github.com/jmigpin/editor/ui/tautil"
	"github.com/jmigpin/editor/uiutil"
	"github.com/jmigpin/editor/xutil/keybmap"
	"github.com/jmigpin/editor/xutil/xgbutil"

	"golang.org/x/image/math/fixed"
)

type TextArea struct {
	C             uiutil.Container
	ui            *UI
	EvReg         *xgbutil.EventRegister
	dereg         xgbutil.EventDeregister
	stringCache   *drawutil.StringCache
	editHistory   *tautil.EditHistory
	edit          *tautil.EditHistoryEdit
	buttonPressed bool
	boundsChange  image.Rectangle

	str         string
	cursorIndex int
	offsetY     fixed.Int26_6
	selection   struct {
		on    bool
		index int // from index to cursorIndex
	}

	Colors                     *drawutil.Colors
	DisableHighlightCursorWord bool
	DisablePageUpDown          bool
}

func NewTextArea(ui *UI) *TextArea {
	ta := &TextArea{ui: ui}
	c := drawutil.DefaultColors()
	ta.Colors = &c
	ta.C.PaintFunc = ta.paint
	ta.C.OnCalcFunc = ta.onContainerCalc
	ta.stringCache = drawutil.NewStringCache(ui.FontFace())
	ta.EvReg = xgbutil.NewEventRegister()
	ta.editHistory = tautil.NewEditHistory(30)

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
func (ta *TextArea) Bounds() *image.Rectangle {
	return &ta.C.Bounds
}
func (ta *TextArea) Error(err error) {
	ta.EvReg.Emit(TextAreaErrorEventId, err)
}

func (ta *TextArea) onContainerCalc() {
	ta.updateStringCacheWithBoundsChangedCheck()
}
func (ta *TextArea) updateStringCacheWithBoundsChangedCheck() {
	// check if bounds have changed to emit event
	changed := false
	offsetIndex := 0
	if !ta.C.Bounds.Eq(ta.boundsChange) {
		changed = true
		ta.boundsChange = ta.C.Bounds
		offsetIndex = ta.OffsetIndex()
	}

	ta.updateStringCache()

	if changed {
		// set offset to keep the same first line while resizing
		ta.SetOffsetIndex(offsetIndex)

		ev := &TextAreaBoundsChangeEvent{ta}
		ta.EvReg.Emit(TextAreaBoundsChangeEventId, ev)
	}
}
func (ta *TextArea) updateStringCache() {
	ta.stringCache.Update(ta.str, ta.C.Bounds.Dx())
}
func (ta *TextArea) StrHeight() fixed.Int26_6 {
	h := ta.stringCache.Height()
	min := ta.LineHeight()
	if h < min {
		h = min
	}
	return h
}

// Used externally for dynamic textarea height.
func (ta *TextArea) CalcStringHeight(width int) int {
	ta.stringCache.Update(ta.str, width)
	return ta.StrHeight().Round()
}

func (ta *TextArea) paint() {
	// fill background
	imageutil.FillRectangle(ta.ui.Image(), &ta.C.Bounds, ta.Colors.Bg)

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

func (ta *TextArea) Str() string {
	if ta.edit != nil {
		// return edit str while editing
		return ta.edit.Str()
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
		ta.editHistory.ClearQ()
		ta.setStr(str)
	} else {
		// replace string, keeping the undo available
		ta.EditOpen()
		ta.EditDelete(0, len(ta.Str()))
		ta.EditInsert(0, str)
		ta.EditClose()
	}
}

func (ta *TextArea) EditOpen() {
	if ta.edit != nil {
		panic("edit already exists")
	}
	ta.edit = tautil.NewEditHistoryEdit(ta.Str())
}
func (ta *TextArea) EditInsert(index int, str string) {
	ta.edit.Insert(index, str)
}
func (ta *TextArea) EditDelete(index, index2 int) {
	ta.edit.Delete(index, index2)
}
func (ta *TextArea) EditClose() {
	str, strEdit, ok := ta.edit.Close()
	ta.edit = nil
	if !ok {
		return
	}
	ta.editHistory.PushEdit(strEdit)
	ta.setStr(str)
}

func (ta *TextArea) popUndo() {
	s, i, ok := ta.editHistory.PopUndo(ta.Str())
	if !ok {
		return
	}
	ta.setStr(s)
	ta.SetCursorIndex(i)
	ta.SetSelectionOn(false)
}
func (ta *TextArea) unpopRedo() {
	s, i, ok := ta.editHistory.UnpopRedo(ta.Str())
	if !ok {
		return
	}
	ta.setStr(s)
	ta.SetCursorIndex(i)
	ta.SetSelectionOn(false)
}

func (ta *TextArea) CursorIndex() int {
	return ta.cursorIndex
}
func (ta *TextArea) SetCursorIndex(v int) {
	if v < 0 {
		v = 0
	}
	if v > len(ta.Str()) {
		v = len(ta.Str())
	}
	if v != ta.cursorIndex {
		ta.cursorIndex = v
		ta.C.NeedPaint()

		ev := &TextAreaSetCursorIndexEvent{ta}
		ta.EvReg.Emit(TextAreaSetCursorIndexEventId, ev)
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
	if v > ta.StrHeight() {
		v = ta.StrHeight()
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
	// is visible
	y0 := ta.OffsetY()
	y1 := y0 + fixed.I(ta.C.Bounds.Dy())
	p0 := ta.stringCache.GetPoint(index).Y
	p1 := p0 + ta.LineHeight()
	if p0 >= y0 && p1 < y1 {
		return
	}
	// set at half bounds
	//p0 := ta.stringCache.GetPoint(index)
	half := fixed.I(ta.C.Bounds.Dy() / 2)
	offsetY := p0 - half
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
func (ta *TextArea) IndexPoint(i int) *fixed.Point26_6 {
	return ta.stringCache.GetPoint(i)
}
func (ta *TextArea) PointIndex(p *fixed.Point26_6) int {
	return ta.stringCache.GetIndex(p)
}

func (ta *TextArea) PageUp() {
	if ta.DisablePageUpDown {
		return
	}
	tautil.PageUp(ta)
}
func (ta *TextArea) PageDown() {
	if ta.DisablePageUpDown {
		return
	}
	tautil.PageDown(ta)
}

func (ta *TextArea) onButtonPress(ev0 xgbutil.EREvent) {
	ev := ev0.(*keybmap.ButtonPressEvent)
	if !ev.Point.In(ta.C.Bounds) {
		return
	}
	ta.buttonPressed = true
	switch {
	case ev.Button.Button1():
		switch {
		case ev.Button.Mods.IsShift():
			tautil.MoveCursorToPoint(ta, ev.Point, true)
		case ev.Button.Mods.IsNone():
			tautil.MoveCursorToPoint(ta, ev.Point, false)
		}
	case ev.Button.Button4():
		canScroll := !ta.DisablePageUpDown
		if canScroll {
			tautil.ScrollUp(ta)
		}
	case ev.Button.Button5():
		canScroll := !ta.DisablePageUpDown
		if canScroll {
			tautil.ScrollDown(ta)
		}
	}
}
func (ta *TextArea) onButtonRelease(ev0 xgbutil.EREvent) {
	if !ta.buttonPressed {
		return
	}
	ta.buttonPressed = false
	ev := ev0.(*keybmap.ButtonReleaseEvent)
	if ev.Button.Mods.IsButton(3) {
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
	if ev.Mods.IsButton(1) {
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
	case keybmap.XKAltL,
		keybmap.XKIsoLevel3Shift,
		keybmap.XKShiftL,
		keybmap.XKShiftR,
		keybmap.XKControlL,
		keybmap.XKControlR,
		keybmap.XKCapsLock,
		keybmap.XKNumLock,
		keybmap.XKSuperL,
		keybmap.XKInsert:
		// ignore these
	case keybmap.XKRight:
		switch {
		case k.Mods.IsControlShift():
			tautil.MoveCursorJumpRight(ta, true)
		case k.Mods.IsControl():
			tautil.MoveCursorJumpRight(ta, false)
		case k.Mods.IsShift():
			tautil.MoveCursorRight(ta, true)
		case k.Mods.IsNone():
			tautil.MoveCursorRight(ta, false)
		}
	case keybmap.XKLeft:
		switch {
		case k.Mods.IsControlShift():
			tautil.MoveCursorJumpLeft(ta, true)
		case k.Mods.IsControl():
			tautil.MoveCursorJumpLeft(ta, false)
		case k.Mods.IsShift():
			tautil.MoveCursorLeft(ta, true)
		case k.Mods.IsNone():
			tautil.MoveCursorLeft(ta, false)
		}
	case keybmap.XKUp:
		switch {
		case k.Mods.IsControlMod1():
			tautil.MoveLineUp(ta)
		case k.Mods.IsShift():
			tautil.MoveCursorUp(ta, true)
		case k.Mods.IsNone():
			tautil.MoveCursorUp(ta, false)
		}
	case keybmap.XKDown:
		switch {
		case k.Mods.IsControlShiftMod1():
			tautil.DuplicateLines(ta)
		case k.Mods.IsControlMod1():
			tautil.MoveLineDown(ta)
		case k.Mods.IsShift():
			tautil.MoveCursorDown(ta, true)
		case k.Mods.IsNone():
			tautil.MoveCursorDown(ta, false)
		}
	case keybmap.XKHome:
		switch {
		case k.Mods.IsControlShift():
			tautil.StartOfString(ta, true)
		case k.Mods.IsControl():
			tautil.StartOfString(ta, false)
		case k.Mods.IsShift():
			tautil.StartOfLine(ta, true)
		case k.Mods.IsNone():
			tautil.StartOfLine(ta, false)
		}
	case keybmap.XKEnd:
		switch {
		case k.Mods.IsControlShift():
			tautil.EndOfString(ta, true)
		case k.Mods.IsControl():
			tautil.EndOfString(ta, false)
		case k.Mods.IsShift():
			tautil.EndOfLine(ta, true)
		case k.Mods.IsNone():
			tautil.EndOfLine(ta, false)
		}
	case keybmap.XKBackspace:
		tautil.Backspace(ta)
	case keybmap.XKDelete:
		switch {
		case k.Mods.IsNone():
			tautil.Delete(ta)
		}
	case keybmap.XKPageUp:
		switch {
		case k.Mods.IsNone():
			ta.PageUp()
		}
	case keybmap.XKPageDown:
		switch {
		case k.Mods.IsNone():
			ta.PageDown()
		}
	case keybmap.XKTab:
		switch {
		case k.Mods.IsNone():
			if ta.SelectionOn() {
				tautil.TabRight(ta)
			} else {
				tautil.InsertRune(ta, '\t')
			}
		case k.Mods.IsShift():
			tautil.TabLeft(ta)
		}
	case keybmap.XKReturn:
		switch {
		case k.Mods.IsNone():
			tautil.InsertRune(ta, '\n')
		}
	case keybmap.XKSpace:
		tautil.InsertRune(ta, ' ')
	default:
		// shortcuts with printable runes
		switch {
		case k.Mods.IsControlShift():
			switch firstKeysym {
			case 'd':
				tautil.Uncomment(ta)
			case 'z':
				ta.unpopRedo()
			}
		case k.Mods.IsControl():
			switch firstKeysym {
			case 'd':
				tautil.Comment(ta)
			case 'c':
				tautil.Copy(ta)
			case 'x':
				tautil.Cut(ta)
			case 'v':
				tautil.Paste(ta)
			case 'k':
				tautil.RemoveLines(ta)
			case 'a':
				tautil.SelectAll(ta)
			case 'z':
				ta.popUndo()
			}
		default: // all other modifier combos
			ta.insertKeyRune(k)
		}
	}
}
func (ta *TextArea) insertKeyRune(k *keybmap.Key) {
	// print rune from keysym table (takes into consideration the modifiers)
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
		tautil.InsertRune(ta, rune(ks))
		// TODO: prevent stringcache calc for just a rune
	}
}

const (
	TextAreaErrorEventId = iota
	TextAreaCmdEventId
	TextAreaSetTextEventId
	TextAreaSetOffsetYEventId
	TextAreaBoundsChangeEventId
	TextAreaSetCursorIndexEventId
)

type TextAreaCmdEvent struct {
	TextArea *TextArea
}
type TextAreaSetTextEvent struct {
	TextArea  *TextArea
	OldBounds image.Rectangle
}
type TextAreaSetOffsetYEvent struct {
	TextArea *TextArea
}
type TextAreaBoundsChangeEvent struct {
	TextArea *TextArea
}
type TextAreaSetCursorIndexEvent struct {
	TextArea *TextArea
}
