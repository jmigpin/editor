package ui

import (
	"image"

	"github.com/jmigpin/editor/uiutil/event"
	"github.com/jmigpin/editor/uiutil/widget"
)

type MainMenuButton struct {
	*widget.Button
	FloatMenu *FloatMenu

	ui *UI
}

func NewMainMenuButton(ui *UI) *MainMenuButton {
	m := &MainMenuButton{ui: ui}
	m.Button = widget.NewButton(ui)
	m.SetWrapper(m)
	m.Button.Label.Text.Str = string(rune(8801)) // 3 lines rune
	m.Button.Label.Pad.Left = 5
	m.Button.Label.Pad.Right = 5
	m.Button.Sticky = true
	m.FloatMenu = NewFloatMenu(m)
	return m
}
func (m *MainMenuButton) CalcChildsBounds() {
	m.EmbedNode.CalcChildsBounds()
	if !m.FloatMenu.Hidden() {
		m.FloatMenu.CalcChildsBounds()
	}
}
func (m *MainMenuButton) OnInputEvent(ev0 interface{}, p image.Point) bool {
	m.Button.OnInputEvent(ev0, p)
	switch ev0.(type) {
	case *event.MouseClick:
		toggle := m.FloatMenu.Hidden()
		m.FloatMenu.ShowCalcMark(toggle)
	}
	return false
}

type FloatMenu struct {
	*widget.FloatBox
	Toolbar *Toolbar

	m *MainMenuButton
}

func NewFloatMenu(m *MainMenuButton) *FloatMenu {
	fm := &FloatMenu{m: m}

	fm.Toolbar = NewToolbar(m.ui, &m.ui.Layout)
	pad := widget.NewPad(m.ui, fm.Toolbar)
	pad.Set(10)
	pad.Color = &fm.Toolbar.Colors.Normal.Bg
	border := widget.NewPad(m.ui, pad)
	border.Set(1)
	border.Color = &fm.Toolbar.Colors.Normal.Fg

	// shadow
	var container widget.Node = border
	if ShadowsOn {
		shadow := widget.NewShadow(m.ui, border)
		shadow.Bottom = ShadowSteps
		shadow.MaxShade = ShadowMaxShade
		container = shadow
	}

	fm.FloatBox = widget.NewFloatBox(container)
	fm.SetWrapper(fm)

	fm.SetHidden(true)

	return fm
}
func (fm *FloatMenu) CalcChildsBounds() {
	b := fm.m.Bounds
	fm.RefPoint = image.Point{b.Min.X, b.Max.Y}
	fm.FloatBox.CalcChildsBounds()
}
func (fm *FloatMenu) OnInputEvent(ev0 interface{}, p image.Point) bool {
	// don't let other layers get the event
	return true
}
