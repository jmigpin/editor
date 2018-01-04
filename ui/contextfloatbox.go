package ui

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type ContextFloatBox struct {
	*widget.FloatBox
	Label   *widget.Label
	layout  *Layout
	Enabled bool // used externally
}

func NewContextFloatBox(l *Layout) *ContextFloatBox {
	cfb := &ContextFloatBox{layout: l}

	cfb.Label = widget.NewLabel(l.UI)
	cfb.Label.Text.Str = "testing"
	cfb.Label.Text.Color = &TextAreaColors.Normal.Fg
	cfb.Label.Color = &TextAreaColors.Normal.Bg
	cfb.Label.Pad.Left = 5
	cfb.Label.Pad.Right = 5
	cfb.Label.Pad.Color = &TextAreaColors.Normal.Bg
	cfb.Label.Border.Set(1)
	cfb.Label.Border.Color = &TextAreaColors.Normal.Fg

	//scrollArea := widget.NewScrollArea(l.UI, cfb.Label)
	////scrollArea.SetExpand(true, true)
	//scrollArea.LeftScroll = ScrollbarLeft
	//scrollArea.ScrollWidth = ScrollbarWidth
	//scrollArea.VBar.Color = &ScrollbarBgColor
	//scrollArea.VBar.Handle.Color = &ScrollbarFgColor

	//border := widget.NewPad(l.UI, scrollArea)
	//border.Set(1)
	//border.Color = &TextAreaColors.Normal.Fg

	// shadow
	var container widget.Node = cfb.Label
	//var container widget.Node = border
	if ShadowsOn {
		shadow := widget.NewShadow(l.UI, container)
		shadow.Bottom = ShadowSteps
		shadow.MaxShade = ShadowMaxShade
		container = shadow
	}

	cfb.FloatBox = widget.NewFloatBox(container)

	cfb.SetHidden(true)

	return cfb
}

func (cfb *ContextFloatBox) OnInputEvent(ev interface{}, p image.Point) bool {
	switch ev.(type) {
	case *event.KeyUp,
		*event.KeyDown:
		// let lower layers get events
		return false
	}
	return true
}
