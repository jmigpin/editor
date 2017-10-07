package widget

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/colornames"
)

type ScrollArea struct {
	EmbedNode

	Fg, Bg      color.Color
	ScrollWidth int
	LeftScroll  bool

	vbar struct {
		sizePercent     float64
		positionPercent float64
		bounds          image.Rectangle
		scrollBounds    image.Rectangle
		origPad         image.Point
	}

	ui UIer
}

func (sa *ScrollArea) Init(uier UIer) {
	*sa = ScrollArea{
		ui:          uier,
		Fg:          colornames.Darkorange,
		Bg:          colornames.Antiquewhite,
		ScrollWidth: 10,
		LeftScroll:  true,
	}
	sa.vbar.positionPercent = 0.0
	sa.vbar.sizePercent = 1.0
}

func (sa *ScrollArea) CalcPosition(offset, height, viewh float64) {
	pp := 0.0
	sp := 1.0
	if height > viewh {
		dh := height - viewh
		if offset < 0 {
			offset = 0
		} else if offset > dh {
			offset = dh
		}
		pp = offset / height
		sp = viewh / height
		if sp > 1 {
			sp = 1
		}
	}
	sa.vbar.sizePercent = sp
	sa.vbar.positionPercent = pp
}

func (sa *ScrollArea) CalcPositionFromPoint(p *image.Point) {
	// Called when dragging the scrollbar

	py := float64(p.Sub(sa.vbar.origPad).Sub(sa.vbar.bounds.Min).Y)
	dy := float64(sa.vbar.bounds.Dy())

	offset := py / dy
	height := 1.0
	viewh := sa.vbar.sizePercent

	sa.CalcPosition(offset, height, viewh)
}

func (sa *ScrollArea) SetVBarOrigPad(p *image.Point) {
	b := sa.vbar.scrollBounds
	if p.In(b) {
		// set position relative to the bar top
		sa.vbar.origPad.X = p.X - b.Min.X
		sa.vbar.origPad.Y = p.Y - b.Min.Y
	} else {
		// set position in the middle of the bar
		sa.vbar.origPad.X = b.Dx() / 2
		sa.vbar.origPad.Y = b.Dy() / 2
	}
}

func (sa *ScrollArea) VBarPositionPercent() float64 {
	return sa.vbar.positionPercent
}
func (sa *ScrollArea) VBarBounds() *image.Rectangle {
	return &sa.vbar.bounds
}

func (sa *ScrollArea) Measure(hint image.Point) image.Point {
	// Not measuring child or a big value could be passed up.
	// A scrollarea allows the child node to be small.

	return image.Point{50, 50}
}

func (sa *ScrollArea) CalcChildsBounds() {
	if sa.NChilds() == 0 {
		panic("!")
	}

	// bar
	sa.vbar.bounds = sa.Bounds()
	vbb := &sa.vbar.bounds
	if sa.LeftScroll {
		vbb.Max.X = vbb.Min.X + sa.ScrollWidth
	} else {
		vbb.Min.X = vbb.Max.X - sa.ScrollWidth
	}

	// scroll
	r2 := *vbb
	r2.Min.Y += int(math.Ceil(float64(vbb.Dy()) * sa.vbar.positionPercent))
	size := int(math.Ceil(float64(vbb.Dy()) * sa.vbar.sizePercent))
	if size < 3 {
		size = 3 // minimum bar size (stay visible)
	}
	r2.Max.Y = r2.Min.Y + size
	r2 = r2.Intersect(*vbb)
	sa.vbar.scrollBounds = r2

	// child bounds
	r := sa.Bounds()
	if sa.LeftScroll {
		r.Min.X = sa.vbar.bounds.Max.X
	} else {
		r.Max.X = sa.vbar.bounds.Min.X
	}
	child := sa.FirstChild()
	child.SetBounds(&r)
	child.CalcChildsBounds()
}

func (sa *ScrollArea) Paint() {
	sa.ui.FillRectangle(&sa.vbar.bounds, sa.Bg)
	sa.ui.FillRectangle(&sa.vbar.scrollBounds, sa.Fg)
}
