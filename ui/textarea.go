package ui

import (
	"image"
	"jmigpin/editor/drawutil"
	"jmigpin/editor/ui/tautil"
	"jmigpin/editor/xutil/keybmap"

	"github.com/BurntSushi/xgb/xproto"

	"golang.org/x/image/math/fixed"
)

type TextArea struct {
	Container
	Colors   *drawutil.Colors
	DynamicY bool // adjust area.y to used area (ex:toolbars)

	stringCache drawutil.StringCache

	text        string
	cursorIndex int
	offsetY     fixed.Int26_6
	selection   struct { // text selection
		on    bool
		index int // starting index
	}
	//partialPaint struct{
	//need bool
	//rect *image.Rectangle
	//offsetY fixed.Int26_6
	//}

	Data interface{} // for external use (ex parent container)

	cache struct {
		offsetIndex struct {
			firstCalcDone bool
			areaDx        int
		}

		undo struct {
			pos  int
			text []string
		}
	}
}

func NewTextArea() *TextArea {
	ta := &TextArea{}
	c := drawutil.DefaultColors()
	ta.Colors = &c
	ta.Container.Painter = ta
	ta.Container.OnPointEvent = ta.onPointEvent
	return ta
}
func (ta *TextArea) CalcArea(area *image.Rectangle) {
	ta.Area = *area

	// keep offset index when area was resized
	fixOffsetY := false
	offsetIndex := 0
	u := &ta.cache.offsetIndex
	if u.firstCalcDone && ta.Area.Dx() != u.areaDx {
		fixOffsetY = true
		u.areaDx = ta.Area.Dx()
		offsetIndex = ta.OffsetIndex()
	}
	// flag the run of calcrunedata (needed)
	u.firstCalcDone = true

	// TODO: improve setting face var, textarea doesn't have the ui.face at newtextarea()
	// Calc string cache
	ta.stringCache.Face = ta.UI.FontFace()
	ta.stringCache.CalcRuneData(ta.Text(), ta.Area.Dx())

	if fixOffsetY {
		ta.SetOffsetIndex(offsetIndex)
	}

	if ta.DynamicY {
		th := ta.TextHeight().Round()
		// minimum height in dynamic mode (ex: empty text)
		lh := ta.LineHeight().Round()
		if th < lh {
			th = lh
		}
		// adjust to the used y
		y := ta.Area.Min.Y + th
		if y < ta.Area.Max.Y {
			ta.Area.Max.Y = y
		}
	}
}
func (ta *TextArea) Paint() {
	//if !ta.partialPaint.need{
	//// fill background
	//drawutil.FillRectangle(ta.UI.RootImage(), &ta.Area, ta.Colors.Bg)
	//}
	// fill background
	drawutil.FillRectangle(ta.UI.RootImage(), &ta.Area, ta.Colors.Bg)

	var selection *drawutil.Selection
	if ta.selection.on {
		selection = &drawutil.Selection{
			StartIndex: ta.selection.index,
			EndIndex:   ta.cursorIndex,
		}
	}

	//// partial painting
	//rect:=&ta.Area
	//offsetY := ta.offsetY
	//if ta.partialPaint.need{
	//u:=ta.Area
	//u.Min.Y = ta.partialPaint.rect.Min.Y
	//u.Max.Y = ta.partialPaint.rect.Max.Y
	////rect:=ta.partialPaint.rect
	//rect=&u
	//offsetY = ta.partialPaint.offsetY
	//ta.clearPartialPaint()
	//}

	// img needs to be clipped for drawing
	img := ta.UI.RootImageSubImage(&ta.Area)

	highlight := ta.DynamicY == false && selection == nil

	ta.stringCache.Draw(
		img,
		ta.cursorIndex,
		ta.offsetY,
		ta.Colors,
		selection,
		highlight)
}

//func (ta*TextArea)NeedPaint(){
//ta.Container.NeedPaint()
//ta.clearPartialPaint()
//}
//func (ta*TextArea) NeedPartialPaint(r*image.Rectangle){
//ta.NeedPaint()
//ta.partialPaint.need = true
//ta.partialPaint.rect=r
//}
//func (ta*TextArea)clearPartialPaint(){
//ta.partialPaint.need=false
//ta.partialPaint.rect = nil
//}

// Implements tautil.texta
func (ta *TextArea) Error(err error) {
	ta.UI.PushEvent(err)
}

func (ta *TextArea) Text() string {
	return ta.text
}
func (ta *TextArea) SetText(v string) {
	if v != ta.text {
		ta.text = v
		oldArea := ta.Area
		ta.CalcOwnArea()
		ta.NeedPaint()
		ta.UI.PushEvent(&TextAreaSetTextEvent{ta, oldArea})
	}
}
func (ta *TextArea) ClearText() {
	ta.cursorIndex = 0
	ta.offsetY = 0
	ta.selection.on = false
	ta.SetText("")
}
func (ta *TextArea) CursorIndex() int {
	return ta.cursorIndex
}
func (ta *TextArea) SetCursorIndex(v int) {
	if v < 0 {
		v = 0
	}
	if v > len(ta.text) {
		v = len(ta.text)
	}
	if v != ta.cursorIndex {
		ta.cursorIndex = v
		ta.NeedPaint()
	}
}
func (ta *TextArea) OffsetY() fixed.Int26_6 {
	return ta.offsetY
}
func (ta *TextArea) SetOffsetY(v fixed.Int26_6) {
	if v != ta.offsetY {
		if v < 0 {
			v = 0
		}
		if v > ta.TextHeight() {
			v = ta.TextHeight()
		}
		ta.offsetY = v
		ta.CalcOwnArea()
		ta.NeedPaint()
		ta.UI.PushEvent(&TextAreaSetOffsetYEvent{ta})
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
	half := fixed.I(ta.Area.Dy() / 2)
	offsetY := p.Y - half
	ta.SetOffsetY(offsetY)
}
func (ta *TextArea) WarpPointerToCursor() {
	p := ta.stringCache.GetPoint(ta.CursorIndex())
	p.Y -= ta.offsetY
	p2 := drawutil.Point266ToPoint(p)
	p3 := p2.Add(ta.Area.Min)
	// add pad
	p3.Y += ta.LineHeight().Round()
	p3.X += 5
	// ensure the cursor is reachable (ex: area small and cursor is drawn outside of it)
	if !p3.In(ta.Area) {
		p3.X = 0
	}
	ta.UI.WarpPointer(&p3)
}

func (ta *TextArea) SelectionOn() bool {
	return ta.selection.on
}
func (ta *TextArea) SetSelectionOn(v bool) {
	if v != ta.selection.on {
		ta.selection.on = v
		ta.NeedPaint()
	}
}
func (ta *TextArea) SelectionIndex() int {
	return ta.selection.index
}
func (ta *TextArea) SetSelectionIndex(v int) {
	if v != ta.selection.index {
		ta.selection.index = v
		ta.NeedPaint()
	}
}
func (ta *TextArea) RequestTreePaint() {
	ta.UI.RequestTreePaint()
}

func (ta *TextArea) RequestClipboardString() (string, error) {
	return ta.UI.XUtil.Paste.Request()
}
func (ta *TextArea) SetClipboardString(v string) {
	ta.UI.XUtil.Copy.Set(v)
}
func (ta *TextArea) LineHeight() fixed.Int26_6 {
	fm := ta.UI.FontFace().Face.Metrics()
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
	p0i := p.Sub(ta.Area.Min)
	p0 := drawutil.PointToPoint266(&p0i)
	p0.Y += ta.offsetY
	return ta.stringCache.GetIndex(p0)
}

func (ta *TextArea) PushTextCopy() {
	a := ta.cache.undo.text
	if len(a) > 0 {
		s := a[len(a)-1]
		if ta.text == s { // last cache entry is equal
			return
		}
	}
	a = append(a, ta.text)
	if len(a) > 10 {
		copy(a, a[1:])
	}
	ta.cache.undo.text = a
}
func (ta *TextArea) PopTextCopy() (string, bool) {
	a := ta.cache.undo.text
	if len(a) == 0 {
		return "", false
	}
	s := a[len(a)-1] // last
	a = a[:len(a)-1] // pop
	ta.cache.undo.text = a
	return s, true // return last
}

func (ta *TextArea) TextHeight() fixed.Int26_6 {
	return ta.stringCache.TextHeight()
}

func (ta *TextArea) onPointEvent(p *image.Point, ev Event) bool {
	switch ev0 := ev.(type) {
	case *KeyPressEvent:
		ta.onKeyPress(ev0.Key)
		// returning false prevents the event from going to another space it the areas get to be calculated (case of dynamicY)
		return false
	case *ButtonPressEvent:
		// register for layout callbacks
		ta.UI.Layout.OnPointEvent = ta.onRootPointEvent

		ta.onButtonPress(p, ev0.Button)
	}
	return true
}
func (ta *TextArea) onRootPointEvent(p *image.Point, ev Event) bool {
	switch ev0 := ev.(type) {
	case *MotionNotifyEvent:
		ta.onRootMotionNotify(p, ev0.Modifiers)
		ta.UI.RequestMotionNotify()
	case *ButtonReleaseEvent:
		// release callbacks
		ta.UI.Layout.OnPointEvent = nil

		// release that started and ended in the area
		if p.In(ta.Area) {
			ta.onButtonRelease(p, ev0.Button)
		}
	}
	return true
}

func (ta *TextArea) onButtonPress(p *image.Point, b *keybmap.Button) {
	switch b.Button {
	case xproto.ButtonIndex1:
		sel := b.Mods.Shift()
		tautil.MoveCursorToPoint(ta, p, sel)
	case xproto.ButtonIndex4:
		if !ta.DynamicY {
			tautil.ScrollUp(ta)
			ta.UI.PushEvent(&TextAreaScrollEvent{ta, true})
		}
	case xproto.ButtonIndex5:
		if !ta.DynamicY {
			tautil.ScrollDown(ta)
			ta.UI.PushEvent(&TextAreaScrollEvent{ta, false})
		}
	}
}
func (ta *TextArea) onRootMotionNotify(p *image.Point, m keybmap.Modifiers) {
	if m.Button1() {
		tautil.MoveCursorToPoint(ta, p, true)
	}
}
func (ta *TextArea) onButtonRelease(p *image.Point, b *keybmap.Button) {
	switch b.Button {
	case xproto.ButtonIndex2:
		tautil.MoveCursorToPoint(ta, p, false)
		ta.UI.PushEvent(&TextAreaCmdEvent{ta})
	}
}
func (ta *TextArea) onKeyPress(k *keybmap.Key) {
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
				tautil.DupLines(ta)
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

				//case 'a':
				//sel := k.Modifiers.Shift()
				//tautil.StartOfLine(ta, sel)
				//case 'e':
				//sel := k.Modifiers.Shift()
				//tautil.EndOfLine(ta, sel)

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

			//p:=ta.stringCache.GetPoint(ta.CursorIndex())
			//p.Y -= ta.offsetY
			//p2:=drawutil.Point266ToPoint(p)
			//p3:=p2.Add(ta.Area.Min).Sub(image.Point{10,10})
			//p4:=p3.Add(image.Point{40,40})
			//var r image.Rectangle
			//r.Min = p3
			//r.Max = p4

			//ta.partialPaint.offsetY = ta.offsetY + p.Y

			tautil.InsertRune(ta, rune(ks))

			// prevent stringcache calcrunedata
			//ta.stringCache.str = ta.Text()

			//ta.NeedPartialPaint(&r)
		}
	}
}
