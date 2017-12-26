package ui

import (
	"image"

	"github.com/jmigpin/editor/uiutil/widget"
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

	// shadow
	var container widget.Node = cfb.Label
	if ShadowsOn {
		shadow := widget.NewShadow(l.UI, container)
		shadow.Bottom = ShadowSteps
		shadow.MaxShade = ShadowMaxShade
		container = shadow
	}

	cfb.FloatBox = widget.NewFloatBox(container)
	cfb.SetWrapper(cfb)

	cfb.SetHidden(true)

	return cfb
}

func (cfb *ContextFloatBox) SetStr(s string) {
	cfb.Label.Text.Str = s
}

func (cfb *ContextFloatBox) OnInputEvent(ev0 interface{}, p image.Point) bool {
	// let lower layers get events
	return false
}
