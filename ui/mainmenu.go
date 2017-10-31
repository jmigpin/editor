package ui

import (
	"image"

	"github.com/jmigpin/editor/uiutil/event"
	"github.com/jmigpin/editor/uiutil/widget"
)

type MainMenuButton struct {
	widget.Button
	FloatMenu FloatMenu

	ui *UI
}

func NewMainMenuButton(ui *UI) *MainMenuButton {
	m := &MainMenuButton{ui: ui}
	m.Button.Init(ui)
	m.SetWrapper(m)
	m.Button.Label.Text.Str = string(rune(8801)) // 3 lines rune
	m.Button.Label.Pad.Left = 5
	m.Button.Label.Pad.Right = 5
	m.Button.Sticky = true
	m.FloatMenu.Init(m)
	return m
}
func (m *MainMenuButton) OnInputEvent(ev0 interface{}, p image.Point) bool {
	m.Button.OnInputEvent(ev0, p)
	switch ev0.(type) {
	case *event.MouseClick:
		fm := &m.FloatMenu
		toggle := !fm.Hidden()
		fm.SetHidden(toggle)
		//if !fm.Hidden() {
		//	fm.CalcChildsBounds()
		//	fm.MarkNeedsPaint()
		//}
	}
	return false
}

type FloatMenu struct {
	widget.FloatBox
	Toolbar *Toolbar

	m *MainMenuButton
}

func (fm *FloatMenu) Init(m *MainMenuButton) {
	*fm = FloatMenu{m: m}

	fm.Toolbar = NewToolbar(m.ui, &m.ui.Layout)
	var pad widget.Pad
	pad.Init(m.ui, fm.Toolbar)
	pad.Set(10)
	pad.Color = &fm.Toolbar.Colors.Normal.Bg
	var border widget.Pad
	border.Init(m.ui, &pad)
	border.Set(1)
	border.Color = &fm.Toolbar.Colors.Normal.Fg

	fm.FloatBox.Init(&m.ui.Layout.MultiLayer, &border)
	fm.SetWrapper(fm)

	fm.SetHidden(true)
}
func (fm *FloatMenu) CalcChildsBounds() {
	b := fm.m.Bounds()
	fm.RefPoint = image.Point{b.Min.X, b.Max.Y}
	fm.FloatBox.CalcChildsBounds()
}
