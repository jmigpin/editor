package ui

import (
	"fmt"
	"image"

	"github.com/jmigpin/editor/uiutil/event"
	"github.com/jmigpin/editor/uiutil/widget"
	"github.com/jmigpin/editor/xgbutil/evreg"
)

type MainMenu struct {
	widget.Button
	ui *UI
}

func NewMainMenu(ui *UI) *MainMenu {
	mm := &MainMenu{ui: ui}
	mm.Button.Init(ui)
	mm.Button.Label.Text.Str = string(rune(8801))
	mm.Button.Label.Pad.Left = 5
	mm.Button.Label.Pad.Right = 5
	return mm
}
func (mm *MainMenu) OnInputEvent(ev0 interface{}, p image.Point) bool {
	mm.Button.OnInputEvent(ev0, p)
	switch ev0.(type) {
	case *event.MouseClick:
		mm.ui.EvReg.Enqueue(evreg.ErrorEventId, fmt.Errorf("TODO"))
	}
	return false
}
