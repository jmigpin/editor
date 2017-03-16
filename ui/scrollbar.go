package ui

import (
	"image"

	"golang.org/x/image/math/fixed"

	"github.com/jmigpin/editor/ui/tautil"
	"github.com/jmigpin/editor/uiutil"
	"github.com/jmigpin/editor/xutil/keybmap"
	"github.com/jmigpin/editor/xutil/xgbutil"
)

// Scrollbar for the Textarea.
type Scrollbar struct {
	C             uiutil.Container
	ta            *TextArea
	buttonPressed bool
	bar           struct { // inner rectangle
		sizePercent     float64
		positionPercent float64
		bounds          image.Rectangle
		origPad         image.Point
	}
	dereg xgbutil.EventDeregister
}

func NewScrollbar(ta *TextArea) *Scrollbar {
	sb := &Scrollbar{ta: ta}
	width := ScrollbarWidth
	sb.C.Style.MainSize = &width
	sb.C.PaintFunc = sb.paint

	r1 := sb.ta.ui.Win.EvReg.Add(keybmap.ButtonPressEventId,
		&xgbutil.ERCallback{sb.onButtonPress})
	r2 := sb.ta.ui.Win.EvReg.Add(keybmap.ButtonReleaseEventId,
		&xgbutil.ERCallback{sb.onButtonRelease})
	r3 := sb.ta.ui.Win.EvReg.Add(keybmap.MotionNotifyEventId,
		&xgbutil.ERCallback{sb.onMotionNotify})
	sb.dereg.Add(r1, r2, r3)

	// textarea set text
	sb.ta.EvReg.Add(TextAreaSetTextEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			sb.calcPositionAndSize()
			sb.C.NeedPaint()
		}})
	// textarea scroll (key based scroll)
	sb.ta.EvReg.Add(TextAreaScrollEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			sb.calcPositionAndSize()
			sb.C.NeedPaint()
		}})
	// textarea y jump
	sb.ta.EvReg.Add(TextAreaSetOffsetYEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			sb.calcPositionAndSize()
			sb.C.NeedPaint()
		}})

	return sb
}
func (sb *Scrollbar) calcPositionAndSize() {
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
	sb.bar.sizePercent = sp
	sb.bar.positionPercent = pp
}

// Dragging the scrollbar
func (sb *Scrollbar) calcPositionFromPoint(p *image.Point) {
	r := sb.C.Bounds
	height := r.Dy()
	py := p.Add(sb.bar.origPad).Y - r.Min.Y
	if py < 0 {
		py = 0
	} else if py > height {
		py = height
	}
	sb.bar.positionPercent = float64(py) / float64(height)
}

//// Scrolling with scroll buttons
//func (sb *Scrollbar) calcPositionFromScroll(up bool) {
//mult := 1.0
//if up {
//mult = -1
//}
//// include last line from previous page
//lh := sb.ta.LineHeight().Round()
//th := sb.ta.TextHeight().Round()
//linep := float64(lh) / float64(th)
//pp := sb.bar.positionPercent + mult*(sb.bar.sizePercent-linep)
//if pp < 0 {
//pp = 0
//} else if pp > 1 {
//pp = 1
//}
//sb.bar.positionPercent = pp
//}

func (sb *Scrollbar) setTextareaOffset() {
	pp := sb.bar.positionPercent
	textHeight := sb.ta.TextHeight()
	py := fixed.I(int(pp * float64(textHeight.Round())))
	sb.ta.SetOffsetY(py)
}

func (sb *Scrollbar) paint() {
	// background
	sb.ta.ui.FillRectangle(&sb.C.Bounds, &ScrollbarBgColor)
	// bar
	r := sb.C.Bounds
	size := int(float64(r.Dy()) * sb.bar.sizePercent)
	if size < 3 { // minimum size
		size = 3
	}
	r2 := r
	r2.Min.Y += int(float64(r.Dy()) * sb.bar.positionPercent)
	r2.Max.Y = r2.Min.Y + size
	r2 = r2.Intersect(sb.C.Bounds)
	sb.ta.ui.FillRectangle(&r2, &ScrollbarFgColor)
	sb.bar.bounds = r2
}
func (sb *Scrollbar) onButtonPress(ev0 xgbutil.EREvent) {
	ev := ev0.(*keybmap.ButtonPressEvent)
	if !ev.Point.In(sb.C.Bounds) {
		return
	}
	sb.buttonPressed = true
	switch {
	case ev.Button.Button1():
		sb.setOrigPad(ev.Point) // keep pad for drag calc
		sb.calcPositionFromPoint(ev.Point)
		sb.C.NeedPaint()
		sb.setTextareaOffset()
	case ev.Button.Button4(): // scroll up
		tautil.PageUp(sb.ta)
		//sb.calcPositionFromScroll(true)
		//sb.C.NeedPaint()
		//sb.setTextareaOffset()
	case ev.Button.Button5(): // scroll down
		tautil.PageDown(sb.ta)
		//sb.calcPositionFromScroll(false)
		//sb.C.NeedPaint()
		//sb.setTextareaOffset()
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
		sb.setTextareaOffset()
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
		sb.setTextareaOffset()
	}
}
func (sb *Scrollbar) setOrigPad(p *image.Point) {
	if p.In(sb.bar.bounds) {
		// set position relative to the bar top
		r := &sb.bar.bounds
		sb.bar.origPad.X = r.Max.X - p.X
		sb.bar.origPad.Y = r.Min.Y - p.Y
	} else {
		// set position in the middle of the bar
		r := &sb.bar.bounds
		sb.bar.origPad.X = r.Dx() / 2
		sb.bar.origPad.Y = -r.Dy() / 2
	}
}
func (sb *Scrollbar) Close() {
	sb.dereg.UnregisterAll()
}
