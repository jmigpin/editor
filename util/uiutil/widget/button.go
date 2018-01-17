package widget

import (
	"image"

	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/uiutil/event"
)

type Button struct {
	EmbedNode
	Label   *Label
	Sticky  bool // stay down after click to behave like a menu button
	OnClick func(*event.MouseClick)

	orig   *Theme // original
	down   bool
	active bool
}

func NewButton(ctx ImageContext) *Button {
	b := &Button{}
	b.Label = NewLabel(ctx)
	b.Append(b.Label)
	return b
}
func (b *Button) OnInputEvent(ev0 interface{}, p image.Point) bool {

	keepColor := func() {
		b.orig = b.Theme // can be nil
	}
	restoreColor := func() {
		b.PropagateTheme(b.orig)
	}
	restoreSwitchedColor := func() {
		p := *b.orig.Palette()
		p.Normal.Fg, p.Normal.Bg = p.Normal.Bg, p.Normal.Fg

		// ensure a theme instance to hold the new pallete
		var t Theme
		if b.orig != nil {
			t = *b.orig
		}

		t.SetPalette(&p)
		b.PropagateTheme(&t)
	}
	hoverShade := func() {
		p := *b.orig.Palette()
		p.Normal.Bg = imageutil.TintOrShade(p.Normal.Bg, 0.10)

		// ensure a theme instance to hold the new pallete
		var t Theme
		if b.orig != nil {
			t = *b.orig
		}

		t.SetPalette(&p)
		b.PropagateTheme(&t)
	}

	switch t := ev0.(type) {
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
	case *event.MouseClick:
		if b.OnClick != nil {
			b.OnClick(t)
		}
	}
	return false
}
