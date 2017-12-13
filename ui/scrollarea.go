package ui

import (
	"github.com/jmigpin/editor/uiutil/widget"
)

type ScrollArea struct {
	widget.ScrollArea
	ta *TextArea
	ui *UI

	disableTextAreaOffsetYEvent bool
}

func NewScrollArea(ui *UI, ta *TextArea) *ScrollArea {
	sa := &ScrollArea{ui: ui, ta: ta}
	sa.ScrollArea.Init(ui, sa, ta)

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
	b := sa.Bounds
	b.Max.X -= sa.ScrollWidth
	_ = sa.ta.Measure(b.Size())

	// calc position using int26_6 values cast to floats
	dy := float64(sa.Bounds.Dy())
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
	oy := int(pp * float64(sa.taHeight()))
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
	lines := int(sa.Bounds.Dy() / line)
	nScrollLines := 4
	if lines < 12 {
		nScrollLines = 1
	}
	v := nScrollLines * mult * line
	sa.setTaOffsetY(sa.ta.OffsetY() + v)

	sa.disableTextAreaOffsetYEvent = false

	sa.CalcChildsBounds()
	sa.MarkNeedsPaint()
}

func (sa *ScrollArea) setTaOffsetY(v int) {
	dy := sa.Bounds.Dy()
	max := sa.taHeight() - dy
	if v > max {
		v = max
	}
	sa.ta.SetOffsetY(v)
}

func (sa *ScrollArea) taHeight() int {
	// extra height allows to scroll past the str height
	dy := sa.Bounds.Dy()
	extra := dy - 2*sa.ta.LineHeight() // keep something visible

	return sa.ta.StrHeight() + extra
}
