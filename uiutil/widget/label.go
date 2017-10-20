package widget

import (
	"image/color"
)

type Label struct {
	ShellEmbedNode
	Text   BasicText
	Border Border
	Pad    Pad
	Bg     color.Color
	ui     UIStrDrawer
}

func (l *Label) Init(ui UIStrDrawer) {
	l.ui = ui
	l.Text.Init(ui)
	l.Pad.Init(&l.Text)
	l.Border.Init(ui, &l.Pad)
	l.Bg = color.White
	AppendChilds(l, &l.Border)
}
func (l *Label) Paint() {
	if l.Bg != nil {
		u := l.Bounds()
		l.ui.FillRectangle(&u, l.Bg)
	}
}
