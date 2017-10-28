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
	Sticky bool

	fg, bg color.Color
	down   bool
	active bool
}

func NewButton(ctx Context) *Button {
	b := &Button{}
	b.SetWrapper(b)
	b.Label = NewLabel(ctx)
	b.Append(b.Label)
	return b
}
func (b *Button) OnInputEvent(ev0 interface{}, p image.Point) bool {

	keepColor := func() {
		b.fg = b.Label.Text.Color
		b.bg = b.Label.Bg
	}
	restoreColor := func() {
		b.Label.Text.Color = b.fg
		b.Label.Bg = b.bg
	}
	restoreSwitchedColor := func() {
		b.Label.Text.Color = b.bg
		b.Label.Bg = b.fg
	}
	hoverShade := func() {
		b.Label.Bg = imageutil.Shade(b.bg, 0.10)
	}

	switch ev0.(type) {
	case *event.MouseEnter:
		if !b.active {
			keepColor()
			hoverShade()
			b.MarkNeedsPaint()
		}
	case *event.MouseLeave:
		if !b.active {
			restoreColor()
			b.MarkNeedsPaint()
		}

	case *event.MouseDown:
		if b.active {

		} else {
			b.down = true
			restoreSwitchedColor()
			b.MarkNeedsPaint()
		}
	case *event.MouseUp:
		if b.down {
			b.down = false
			if b.Sticky {
				b.active = true
			} else {
				restoreColor()
				hoverShade()
				b.MarkNeedsPaint()
			}
		} else if b.active {
			b.active = false
			restoreColor()
			hoverShade()
			b.MarkNeedsPaint()
		}
	}
	return false
}
