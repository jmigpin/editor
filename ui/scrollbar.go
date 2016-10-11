package ui

import (
	"image"

	"golang.org/x/image/math/fixed"

	"github.com/BurntSushi/xgb/xproto"
)

// Scrollbar for the Textarea.
type Scrollbar struct {
	Container
	textArea *TextArea

	SizePercent     float64 // bar size
	PositionPercent float64 // bar position

	paddedArea image.Rectangle
	bar        struct { // inner rectangle
		area    image.Rectangle
		origPad image.Point
	}
}

func NewScrollbar(ta *TextArea) *Scrollbar {
	sb := &Scrollbar{textArea: ta}
	sb.Container.Painter = sb
	sb.Container.OnPointEvent = sb.onPointEvent
	return sb
}
func (sb *Scrollbar) CalcArea(area *image.Rectangle) {
	sb.Area = *area

	// size and position percent (from textArea)
	ta := sb.textArea
	sp := 1.0
	pp := 0.0
	textHeight := ta.TextHeight().Round()
	if textHeight > 0 {
		sp = float64(ta.Area.Dy()) / float64(textHeight)
		if sp > 1 {
			sp = 1
		}
		y := sb.textArea.OffsetY().Round()
		pp = float64(y) / float64(textHeight)
	}
	sb.SizePercent = sp
	sb.PositionPercent = pp
}

// Called when dragging the scrollbar.
func (sb *Scrollbar) calcPositionFromPoint(p *image.Point) {
	pa := sb.paddedArea
	height := pa.Dy()
	py := p.Add(sb.bar.origPad).Y - pa.Min.Y
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
	lh := sb.textArea.LineHeight().Round()
	th := sb.textArea.TextHeight().Round()
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
	textHeight := sb.textArea.TextHeight()
	py := fixed.I(int(pp * float64(textHeight.Round())))
	sb.textArea.SetOffsetY(py)
}
func (sb *Scrollbar) Paint() {
	sb.FillRectangle(&sb.Area, &ScrollbarBgColor)
	// padding
	r := sb.Area
	pad := 0
	r.Min.X += pad
	r.Max.X -= pad
	r.Min.Y += pad
	r.Max.Y -= pad
	sb.paddedArea = r.Intersect(sb.Area)
	// bar
	r2 := sb.paddedArea
	size := int(float64(sb.paddedArea.Dy()) * sb.SizePercent)
	if size < 2 {
		size = 2
	}
	r2.Min.Y += int(float64(sb.paddedArea.Dy()) * sb.PositionPercent)
	r2.Max.Y = r2.Min.Y + size
	r2 = r2.Intersect(sb.Area)
	sb.FillRectangle(&r2, &ScrollbarFgColor)
	sb.bar.area = r2
}
func (sb *Scrollbar) onPointEvent(p *image.Point, ev Event) bool {
	switch ev0 := ev.(type) {
	case *ButtonPressEvent:
		// register for layout callbacks
		sb.UI.Layout.OnPointEvent = sb.onRootPointEvent

		switch ev0.Button.Button {
		case xproto.ButtonIndex1:
			sb.updateOrigPad(p) // keep pad for drag calc
			sb.calcPositionFromPoint(p)
			sb.NeedPaint()
			sb.calcTextareaOffset()
		case xproto.ButtonIndex4:
			sb.calcPositionFromScroll(true) // scroll up
			sb.NeedPaint()
			sb.calcTextareaOffset()
		case xproto.ButtonIndex5:
			sb.calcPositionFromScroll(false) // scroll down
			sb.NeedPaint()
			sb.calcTextareaOffset()
		}
	}
	return true
}
func (sb *Scrollbar) onRootPointEvent(p *image.Point, ev Event) bool {
	switch ev0 := ev.(type) {
	case *ButtonReleaseEvent:
		// release callback
		sb.UI.Layout.OnPointEvent = nil

		if ev0.Button.Button1() {
			sb.calcPositionFromPoint(p)
			sb.NeedPaint()
			sb.calcTextareaOffset()
		}
	case *MotionNotifyEvent:
		if ev0.Modifiers.Button1() {
			sb.calcPositionFromPoint(p)
			sb.NeedPaint()
			sb.calcTextareaOffset()
		}
		sb.UI.RequestMotionNotify()
	}
	return true
}
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
