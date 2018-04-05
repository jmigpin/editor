package widget

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/uiutil/event"
)

type Button struct {
	EmbedNode
	Label   *Label
	Sticky  bool // stay down after click to behave like a menu button
	OnClick func(*event.MouseClick)

	down    bool
	sticked bool

	orig struct { // original
		th *Theme
		//pal    Palette
		fg, bg color.Color
	}
}

func NewButton(ctx ImageContext) *Button {
	b := &Button{}
	b.Label = NewLabel(ctx)
	b.Append(b.Label)
	return b
}
func (b *Button) OnInputEvent(ev0 interface{}, p image.Point) bool {
	keepColor := func() {
		b.orig.th = b.Theme()
		b.orig.fg = b.TreeThemePaletteColor("fg")
		b.orig.bg = b.TreeThemePaletteColor("bg")
	}
	restoreColor := func() {
		b.SetTheme(b.orig.th)
	}
	restoreSwitchedColor := func() {
		t := b.orig.th.Copy()
		t.Palette["fg"] = b.orig.bg
		t.Palette["bg"] = b.orig.fg
		b.SetTheme(t)
	}
	hoverShade := func() {
		t := b.orig.th.Copy()
		t.Palette["bg"] = imageutil.TintOrShade(b.orig.bg, 0.10)
		b.SetTheme(t)
	}

	switch t := ev0.(type) {
	case *event.MouseEnter:
		if !b.sticked {
			keepColor()
			hoverShade()
			b.MarkNeedsPaint()
		}
	case *event.MouseLeave:
		if !b.sticked {
			restoreColor()
			b.MarkNeedsPaint()
		}

	case *event.MouseDown:
		if !b.sticked {
			b.down = true
			restoreSwitchedColor()
			b.MarkNeedsPaint()
		}
	case *event.MouseUp:
		if b.down {
			b.down = false
			if b.Sticky {
				b.sticked = true
			} else {
				restoreColor()
				hoverShade()
				b.MarkNeedsPaint()
			}
		} else if b.sticked {
			b.sticked = false
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
