package ui

import (
	"image"

	"golang.org/x/image/math/fixed"

	"github.com/jmigpin/editor/uiutil/widget"
	"github.com/jmigpin/editor/xgbutil/xinput"
)

type ScrollArea struct {
	widget.ScrollArea
	ui            *UI
	ta            *TextArea
	buttonPressed bool

	disableTextAreaOffsetYEvent bool
}

func NewScrollArea(ui *UI, ta *TextArea) *ScrollArea {
	sa := &ScrollArea{ui: ui, ta: ta}

	sa.ScrollArea.Init(ui)
	widget.AppendChilds(sa, ta)

	// textarea set text
	sa.ta.EvReg.Add(TextAreaSetStrEventId, func(ev0 interface{}) {
		sa.CalcChildsBounds()
		sa.MarkNeedsPaint()
	})
	// textarea set offset y
	sa.ta.EvReg.Add(TextAreaSetOffsetYEventId, func(ev0 interface{}) {
		if !sa.disableTextAreaOffsetYEvent {
			sa.CalcChildsBounds()
			sa.MarkNeedsPaint()
		}
	})

	return sa
}

func (sa *ScrollArea) CalcChildsBounds() {
	// measure textarea to have accurate str height
	// TODO: needs improvement, using scrollwidth from widget.scrollarea
	b := sa.Bounds()
	b.Max.X -= sa.ScrollWidth
	_ = sa.ta.Measure(b.Sub(b.Min).Max)

	// calc position using int26_6 values cast to floats
	dy := float64(fixed.I(sa.Bounds().Dy()))
	offset := float64(sa.ta.OffsetY())
	height := float64(sa.taHeight())
	sa.ScrollArea.CalcPosition(offset, height, dy)

	sa.ScrollArea.CalcChildsBounds()
}

func (sa *ScrollArea) CalcPositionFromPoint(p *image.Point) {
	// Dragging the scrollbar, updates textarea offset

	sa.ScrollArea.CalcPositionFromPoint(p)

	sa.disableTextAreaOffsetYEvent = true // ignore loop event

	// set textarea offset
	pp := sa.VBarPositionPercent()
	oy := fixed.Int26_6(pp * float64(sa.taHeight()))
	sa.setTaOffsetY(oy)

	sa.disableTextAreaOffsetYEvent = false

	sa.CalcChildsBounds()
	sa.MarkNeedsPaint()
}

func (sa *ScrollArea) CalcPositionFromScroll(up bool) {
	mult := 1
	if up {
		mult = -1
	}

	sa.disableTextAreaOffsetYEvent = true // ignore loop event

	// set textarea offset
	scrollLines := 4
	v := fixed.Int26_6(scrollLines*mult) * sa.ta.LineHeight()
	sa.setTaOffsetY(sa.ta.OffsetY() + v)

	sa.disableTextAreaOffsetYEvent = false

	sa.CalcChildsBounds()
	sa.MarkNeedsPaint()
}

func (sa *ScrollArea) setTaOffsetY(v fixed.Int26_6) {
	dy := fixed.I(sa.Bounds().Dy())
	max := sa.taHeight() - dy
	if v > max {
		v = max
	}
	sa.ta.SetOffsetY(v)
}

func (sa *ScrollArea) taHeight() fixed.Int26_6 {
	// extra height allows to scroll past the str height
	dy := fixed.I(sa.Bounds().Dy())
	extra := dy - 2*sa.ta.LineHeight() // keep something visible

	return sa.ta.StrHeight() + extra
}

func (sa *ScrollArea) OnInputEvent(ev0 interface{}, p image.Point) bool {
	switch evt := ev0.(type) {
	case *xinput.ButtonPressEvent:
		sa.onButtonPress(evt)
	case *xinput.ButtonReleaseEvent:
		sa.onButtonRelease(evt)
	case *xinput.MotionNotifyEvent:
		sa.onMotionNotify(evt)
	}
	return false
}
func (sa *ScrollArea) onButtonPress(ev *xinput.ButtonPressEvent) {
	// allow scrolling in content area
	if ev.Point.In(sa.Bounds()) && !ev.Point.In(*sa.VBarBounds()) {
		switch {
		case ev.Button.Button(4): // wheel up
			sa.CalcPositionFromScroll(true)
		case ev.Button.Button(5): // wheel down
			sa.CalcPositionFromScroll(false)
		}
		return
	}

	if !ev.Point.In(*sa.VBarBounds()) {
		return
	}
	sa.buttonPressed = true
	sa.ui.Layout.SetInputEventNode(sa, true)
	switch {
	case ev.Button.Button(1):
		sa.SetVBarOrigPad(ev.Point) // keep pad for drag calc
		sa.CalcPositionFromPoint(ev.Point)
	case ev.Button.Button(4): // wheel up
		sa.ta.PageUp()
	case ev.Button.Button(5): // wheel down
		sa.ta.PageDown()
	}
}
func (sa *ScrollArea) onButtonRelease(ev *xinput.ButtonReleaseEvent) {
	if !sa.buttonPressed {
		return
	}
	sa.buttonPressed = false
	sa.ui.Layout.SetInputEventNode(sa, false)
	if ev.Button.Button(1) {
		sa.CalcPositionFromPoint(ev.Point)
	}
}
func (sa *ScrollArea) onMotionNotify(ev *xinput.MotionNotifyEvent) {
	if !sa.buttonPressed {
		return
	}
	switch {
	case ev.Mods.HasButton(1):
		sa.CalcPositionFromPoint(ev.Point)
	}
}
