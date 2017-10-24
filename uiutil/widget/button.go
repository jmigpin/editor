package widget

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/imageutil"
	"github.com/jmigpin/editor/uiutil/event"
)

type Button struct {
	ShellEmbedNode
	Label  *Label
	fg, bg color.Color
	down   bool
}

func NewButton(ctx Context) *Button {
	b := &Button{}
	b.SetWrapper(b)
	b.Label = NewLabel(ctx)
	b.Append(b.Label)
	return b
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
		b.down = true
		b.Label.Bg = b.fg
		b.Label.Text.Color = b.bg
		b.MarkNeedsPaint()
	case *event.MouseUp:
		if b.down {
			b.Label.Text.Color = b.fg
			b.Label.Bg = imageutil.Shade(b.bg, 0.10)
			b.MarkNeedsPaint()
		}
	}
	return false
}
