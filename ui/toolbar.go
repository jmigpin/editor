package ui

import "github.com/jmigpin/editor/util/iout/iorw"

type Toolbar struct {
	*TextArea
	warpPointerOnNextLayout bool
}

func NewToolbar(ui *UI) *Toolbar {
	tb := &Toolbar{}
	tb.TextArea = NewTextArea(ui)
	tb.SetThemePaletteNamePrefix("toolbar_")
	tb.RWEvReg.Add(iorw.RWEvIdWrite, tb.onTaWrite)
	return tb
}

func (tb *Toolbar) onTaWrite(ev0 interface{}) {
	// keep pointer inside toolbar
	p, err := tb.ui.QueryPointer()
	if err == nil && p.In(tb.Bounds) {
		tb.warpPointerOnNextLayout = true
		tb.MarkNeedsLayout()
	}
}

func (tb *Toolbar) Layout() {
	tb.TextArea.Layout()

	// warp pointer to inside the toolbar
	if tb.warpPointerOnNextLayout {
		tb.warpPointerOnNextLayout = false
		p, err := tb.ui.QueryPointer()
		if err == nil && !p.In(tb.Bounds) {
			tb.ui.WarpPointerToRectanglePad(tb.Bounds)
		}
	}
}
