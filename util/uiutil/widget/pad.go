package widget

import (
	"image"

	"github.com/jmigpin/editor/util/imageutil"
)

type Pad struct {
	*Padder
}

func NewPad(ctx ImageContext, child Node) *Pad {
	b := &Pad{Padder: NewPadder(ctx, child)}
	b.Padder.colorName = "pad"
	return b
}

//----------

// Used by Pad and Border.
type Padder struct {
	ENode
	Top, Right, Bottom, Left int
	ctx                      ImageContext
	colorName                string
}

func NewPadder(ctx ImageContext, child Node) *Padder {
	p := &Padder{ctx: ctx}
	p.Append(child)
	return p
}

func (p *Padder) Set(t, r, b, l int) {
	p.Top = t
	p.Right = r
	p.Bottom = b
	p.Left = l
}
func (p *Padder) Set2(v [4]int) {
	p.Set(v[0], v[1], v[2], v[3])
}
func (p *Padder) SetAll(v int) {
	p.Set(v, v, v, v)
}

func (p *Padder) Measure(hint image.Point) image.Point {
	h := hint
	h.X -= p.Right + p.Left
	h.Y -= p.Top + p.Bottom
	h = imageutil.MaxPoint(h, image.Point{0, 0})
	m := p.ENode.Measure(h)
	m.X += p.Right + p.Left
	m.Y += p.Top + p.Bottom
	m = imageutil.MinPoint(m, hint)
	return m
}

func (p *Padder) Layout() {
	u := p.Bounds
	u.Min = u.Min.Add(image.Point{p.Left, p.Top})
	u.Max = u.Max.Sub(image.Point{p.Right, p.Bottom})
	u = u.Intersect(p.Bounds)
	p.Iterate2(func(c *EmbedNode) {
		c.Bounds = u
	})
}
func (p *Padder) Paint() {
	c1 := p.TreeThemePaletteColor(p.colorName)

	b := p.Bounds
	// top
	u := b
	u.Max.Y = u.Min.Y + p.Top
	u = u.Intersect(b)
	imageutil.FillRectangle(p.ctx.Image(), u, c1)
	// bottom
	u = b
	u.Min.Y = u.Max.Y - p.Bottom
	u = u.Intersect(b)
	imageutil.FillRectangle(p.ctx.Image(), u, c1)
	// right
	u = b
	u.Min.X = u.Max.X - p.Right
	u = u.Intersect(b)
	imageutil.FillRectangle(p.ctx.Image(), u, c1)
	// left
	u = b
	u.Max.X = u.Min.X + p.Left
	u = u.Intersect(b)
	imageutil.FillRectangle(p.ctx.Image(), u, c1)
}
