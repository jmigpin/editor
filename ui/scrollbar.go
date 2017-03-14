package ui

import (
	"image"

	"golang.org/x/image/math/fixed"

	"github.com/jmigpin/editor/uiutil"
	"github.com/jmigpin/editor/xutil/keybmap"
	"github.com/jmigpin/editor/xutil/xgbutil"
)

// Scrollbar for the Textarea.
type Scrollbar struct {
	C               uiutil.Container
	ta              *TextArea
	buttonPressed   bool
	SizePercent     float64 // bar size
	PositionPercent float64 // bar position

	//paddedArea image.Rectangle
	bar struct { // inner rectangle
		area    image.Rectangle
		origPad image.Point
	}
}

func NewScrollbar(ta *TextArea) *Scrollbar {
	sb := &Scrollbar{ta: ta}
	width := ScrollbarWidth
	sb.C.Style.MainSize = &width
	sb.C.PaintFunc = sb.paint

	fn := &xgbutil.ERCallback{sb.onButtonPress}
	sb.ta.ui.Win.EvReg.Add(keybmap.ButtonPressEventId, fn)
	fn = &xgbutil.ERCallback{sb.onButtonRelease}
	sb.ta.ui.Win.EvReg.Add(keybmap.ButtonReleaseEventId, fn)
	fn = &xgbutil.ERCallback{sb.onMotionNotify}
	sb.ta.ui.Win.EvReg.Add(keybmap.MotionNotifyEventId, fn)

	return sb
}
func (sb *Scrollbar) CalcArea(area *image.Rectangle) {
	sb.C.Bounds = *area

	// size and position percent (from textArea)
	ta := sb.ta
	sp := 1.0
	pp := 0.0
	textHeight := ta.TextHeight().Round()
	if textHeight > 0 {
		sp = float64(ta.C.Bounds.Dy()) / float64(textHeight)
		if sp > 1 {
			sp = 1
		}
		y := sb.ta.OffsetY().Round()
		pp = float64(y) / float64(textHeight)
	}
	sb.SizePercent = sp
	sb.PositionPercent = pp
}

// Called when dragging the scrollbar.
func (sb *Scrollbar) calcPositionFromPoint(p *image.Point) {
	r := sb.C.Bounds
	height := r.Dy()
	py := p.Add(sb.bar.origPad).Y - r.Min.Y
	if py < 0 {
		py = 0
	} else if py > height {
		py = height
	}
	sb.PositionPercent = float64(py) / float64(height)
}
func (sb *Scrollbar) calcPositionFromScroll(up bool) {
	mult := 1.0
	if up {
		mult = -1
	}
	// include last line from previous page
	lh := sb.ta.LineHeight().Round()
	th := sb.ta.TextHeight().Round()
	linep := float64(lh) / float64(th)
	pp := sb.PositionPercent + mult*(sb.SizePercent-linep)
	if pp < 0 {
		pp = 0
	} else if pp > 1 {
		pp = 1
	}
	sb.PositionPercent = pp
}

// Called when the position has changed
func (sb *Scrollbar) calcTextareaOffset() {
	pp := sb.PositionPercent
	textHeight := sb.ta.TextHeight()
	py := fixed.I(int(pp * float64(textHeight.Round())))
	sb.ta.SetOffsetY(py)
}
func (sb *Scrollbar) paint() {
	// background
	sb.ta.ui.FillRectangle(&sb.C.Bounds, &ScrollbarBgColor)
	// bar
	r := sb.C.Bounds
	r2 := r
	size := int(float64(r.Dy()) * sb.SizePercent)
	if size < 3 {
		size = 3
	}
	r2.Min.Y += int(float64(r.Dy()) * sb.PositionPercent)
	r2.Max.Y = r2.Min.Y + size
	r2 = r2.Intersect(sb.C.Bounds)
	sb.ta.ui.FillRectangle(&r2, &ScrollbarFgColor)
	sb.bar.area = r2
}
func (sb *Scrollbar) onButtonPress(ev0 xgbutil.EREvent) {
	ev := ev0.(*keybmap.ButtonPressEvent)
	if !ev.Point.In(sb.C.Bounds) {
		return
	}
	sb.buttonPressed = true
	switch {
	case ev.Button.Button1():
		sb.updateOrigPad(ev.Point) // keep pad for drag calc
		sb.calcPositionFromPoint(ev.Point)
		sb.C.NeedPaint()
		sb.calcTextareaOffset()
	case ev.Button.Button4():
		sb.calcPositionFromScroll(true) // scroll up
		sb.C.NeedPaint()
		sb.calcTextareaOffset()
	case ev.Button.Button5():
		sb.calcPositionFromScroll(false) // scroll down
		sb.C.NeedPaint()
		sb.calcTextareaOffset()
	}
}
func (sb *Scrollbar) onMotionNotify(ev0 xgbutil.EREvent) {
	if !sb.buttonPressed {
		return
	}
	ev := ev0.(*keybmap.MotionNotifyEvent)
	switch {
	case ev.Modifiers.Button1():
		sb.calcPositionFromPoint(ev.Point)
		sb.C.NeedPaint()
		sb.calcTextareaOffset()
	}
	sb.ta.ui.RequestMotionNotify()
}
func (sb *Scrollbar) onButtonRelease(ev0 xgbutil.EREvent) {
	if !sb.buttonPressed {
		return
	}
	sb.buttonPressed = false
	ev := ev0.(*keybmap.ButtonReleaseEvent)
	switch {
	case ev.Button.Button1():
		sb.calcPositionFromPoint(ev.Point)
		sb.C.NeedPaint()
		sb.calcTextareaOffset()
	}
}

//func (sb *Scrollbar) onPointEvent(p *image.Point, ev Event) bool {
//switch ev0 := ev.(type) {
//case *ButtonPressEvent:
//// register for layout callbacks
////sb.UI.Layout.OnPointEvent = sb.onRootPointEvent

//}
//return true
//}
//func (sb *Scrollbar) onRootPointEvent(p *image.Point, ev Event) bool {
//switch ev0 := ev.(type) {
////case *ButtonReleaseEvent:
////// release callback
//////sb.UI.Layout.OnPointEvent = nil

////if ev0.Button.Button1() {
////sb.calcPositionFromPoint(p)
////sb.C.NeedPaint()
////sb.calcTextareaOffset()
////}
//case *MotionNotifyEvent:
//if ev0.Modifiers.Button1() {
//sb.calcPositionFromPoint(p)
//sb.C.NeedPaint()
//sb.calcTextareaOffset()
//}
//sb.ta.ui.RequestMotionNotify()
//}
//return true
//}
func (sb *Scrollbar) updateOrigPad(p *image.Point) {
	if p.In(sb.bar.area) {
		// set position relative to the bar top
		a := sb.bar.area
		sb.bar.origPad.X = a.Max.X - p.X
		sb.bar.origPad.Y = a.Min.Y - p.Y
	} else {
		// set position in the middle of the bar
		a := sb.bar.area
		sb.bar.origPad.X = a.Dx() / 2
		sb.bar.origPad.Y = -a.Dy() / 2
	}
}
