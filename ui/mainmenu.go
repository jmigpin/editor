package ui

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type MainMenuButton struct {
	*widget.Button
	FloatMenu *FloatMenu

	ui *UI
}

func NewMainMenuButton(ui *UI) *MainMenuButton {
	m := &MainMenuButton{ui: ui}
	m.Button = widget.NewButton(ui)
	m.Button.PropagateTheme(&UITheme.ToolbarTheme)
	m.Button.Label.Text.SetStr(string(rune(8801))) // 3 lines rune
	m.Button.Label.Pad.Left = 5
	m.Button.Label.Pad.Right = 5
	m.Button.Sticky = true
	m.Button.OnClick = func(*event.MouseClick) {
		toggle := m.FloatMenu.Hidden()
		m.FloatMenu.ShowCalcMark(toggle)
	}

	m.FloatMenu = NewFloatMenu(m)
	return m
}
func (m *MainMenuButton) CalcChildsBounds() {
	m.Button.CalcChildsBounds()
	if !m.FloatMenu.Hidden() {
		m.FloatMenu.CalcChildsBounds()
	}
}

type FloatMenu struct {
	*widget.FloatBox
	Toolbar *Toolbar

	m *MainMenuButton
}

func NewFloatMenu(m *MainMenuButton) *FloatMenu {
	fm := &FloatMenu{m: m}

	fm.Toolbar = NewToolbar(m.ui, m.ui.Root)
	pad := widget.NewPad(m.ui, fm.Toolbar)
	pad.Set(10)
	border := widget.NewBorder(m.ui, pad)
	border.Set(1)
	border.PropagateTheme(&UITheme.ToolbarTheme)

	container := WrapInShadowBottom(m.ui, border)
	fm.FloatBox = widget.NewFloatBox(container)

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
