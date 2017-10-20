package ui

import (
	"golang.org/x/image/math/fixed"

	"github.com/jmigpin/editor/uiutil/widget"
)

type ScrollArea struct {
	widget.ScrollArea
	ui *UI
	ta *TextArea

	disableTextAreaOffsetYEvent bool
}

func NewScrollArea(ui *UI, ta *TextArea) *ScrollArea {
	sa := &ScrollArea{ui: ui, ta: ta}

	sa.ScrollArea.Init(sa, ui, ta)

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

func (sa *ScrollArea) UpdatePositionFromPoint() {
	// Dragging the scrollbar, updates textarea offset

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
	line := sa.ta.LineHeight()
	lines := int(fixed.I(sa.Bounds().Dy()) / line)
	nScrollLines := 4
	if lines < 12 {
		nScrollLines = 1
	}
	v := fixed.Int26_6(nScrollLines*mult) * line
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
