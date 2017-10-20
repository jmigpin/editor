package widget

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/imageutil"
	"github.com/jmigpin/editor/uiutil/event"
)

type Button struct {
	ShellEmbedNode
	Label  Label
	fg, bg color.Color
}

//func NewButton(ui UIStrDrawer) *Button {
//	var b Button
//	b.Init(ui)
//	return &b
//}
func (b *Button) Init(ui UIStrDrawer) {
	b.Label.Init(ui)
	AppendChilds(b, &b.Label)
}
func (b *Button) OnInputEvent(ev0 interface{}, p image.Point) bool {
	switch ev0.(type) {
	case *event.MouseEnter:
		b.fg = b.Label.Text.Color
		b.bg = b.Label.Bg
		b.Label.Bg = imageutil.Shade(b.bg, 0.10)
		b.MarkNeedsPaint()
	case *event.MouseLeave:
		b.Label.Text.Color = b.fg
		b.Label.Bg = b.bg
		b.MarkNeedsPaint()

	case *event.MouseDown:
		b.Label.Bg = b.fg
		b.Label.Text.Color = b.bg
		b.MarkNeedsPaint()
	case *event.MouseUp:
		b.Label.Text.Color = b.fg
		b.Label.Bg = imageutil.Shade(b.bg, 0.10)
		b.MarkNeedsPaint()

	}
	return false
}
