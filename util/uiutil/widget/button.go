package widget

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/event"
)

type Button struct {
	ENode
	Label   *Label
	Sticky  bool // stay down after click to behave like a menu button
	OnClick func(*event.MouseClick)

	down  bool
	stuck bool
}

func NewButton(ctx ImageContext) *Button {
	b := &Button{}
	b.Label = NewLabel(ctx)
	b.Append(b.Label)
	return b
}
func (b *Button) OnInputEvent(ev0 any, p image.Point) event.Handled {
	// set "text_*" one level below (b.Label) to allow subclassing elements (ex: floatbutton) to set their own "text_*" values without disrupting the hover/down/sticky colors.
	restoreColor := func() {
		b.Label.SetThemePaletteColor("text_fg", nil)
		b.Label.SetThemePaletteColor("text_bg", nil)
	}
	hoverShade := func() {
		fg := b.TreeThemePaletteColor("button_hover_fg")
		bg := b.TreeThemePaletteColor("button_hover_bg")
		b.Label.SetThemePaletteColor("text_fg", fg)
		b.Label.SetThemePaletteColor("text_bg", bg)
	}
	downShade := func() {
		fg := b.TreeThemePaletteColor("button_down_fg")
		bg := b.TreeThemePaletteColor("button_down_bg")
		b.Label.SetThemePaletteColor("text_fg", fg)
		b.Label.SetThemePaletteColor("text_bg", bg)
	}
	stickyShade := func() {
		fg := b.TreeThemePaletteColor("button_sticky_fg")
		bg := b.TreeThemePaletteColor("button_sticky_bg")
		b.Label.SetThemePaletteColor("text_fg", fg)
		b.Label.SetThemePaletteColor("text_bg", bg)
	}

	switch t := ev0.(type) {
	case *event.MouseEnter:
		if !b.stuck {
			hoverShade()
		}
	case *event.MouseLeave:
		if !b.stuck {
			restoreColor()
		}
	case *event.MouseDown:
		b.down = true
		if !b.stuck {
			downShade()
		}
	case *event.MouseUp:
		if b.down && !b.stuck {
			hoverShade()
		}
		b.down = false
	case *event.MouseClick:
		if b.Sticky {
			if !b.stuck {
				b.stuck = true
				stickyShade()
			} else {
				b.stuck = false
				hoverShade()
			}
		}
		if b.OnClick != nil {
			b.OnClick(t)
		}
	}
	return false
}
