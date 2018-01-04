package widget

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/uiutil/event"
)

type Button struct {
	EmbedNode
	Label  *Label
	Sticky bool

	fg, bg *color.Color
	down   bool
	active bool
}

func NewButton(ctx Context) *Button {
	b := &Button{}
	b.Label = NewLabel(ctx)
	b.Append(b.Label)
	return b
}
func (b *Button) OnInputEvent(ev0 interface{}, p image.Point) bool {

	keepColor := func() {
		b.fg = b.Label.Text.Color
		b.bg = b.Label.Color
	}
	restoreColor := func() {
		b.Label.Text.Color = b.fg
		b.Label.Color = b.bg
	}
	restoreSwitchedColor := func() {
		b.Label.Text.Color = b.bg
		b.Label.Color = b.fg
	}
	hoverShade := func() {
		var c color.Color = imageutil.Shade(*b.bg, 0.10)
		b.Label.Color = &c
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
