package widget

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/util/imageutil"
)

type Pad struct {
	EmbedNode
	Top, Right, Bottom, Left int
	ctx                      ImageContext
}

func NewPad(ctx ImageContext, child Node) *Pad {
	p := &Pad{ctx: ctx}
	p.Append(child)
	return p
}

func (p *Pad) Set(t, r, b, l int) {
	p.Top = t
	p.Right = r
	p.Bottom = b
	p.Left = l
}
func (p *Pad) Set2(v [4]int) {
	p.Set(v[0], v[1], v[2], v[3])
}
func (p *Pad) SetAll(v int) {
	p.Set(v, v, v, v)
}

func (p *Pad) Measure(hint image.Point) image.Point {
	h := hint
	h.X -= p.Right + p.Left
	h.Y -= p.Top + p.Bottom
	h = imageutil.MaxPoint(h, image.Point{0, 0})
	m := p.EmbedNode.Measure(h)
	m.X += p.Right + p.Left
	m.Y += p.Top + p.Bottom
	m = imageutil.MinPoint(m, hint)
	return m
}
func (p *Pad) CalcChildsBounds() {
	u := p.Bounds
	u.Min = u.Min.Add(image.Point{p.Left, p.Top})
	u.Max = u.Max.Sub(image.Point{p.Right, p.Bottom})
	u = u.Intersect(p.Bounds)
	p.IterChilds(func(c Node) {
		c.Embed().Bounds = u
		c.CalcChildsBounds()
	})
}
func (p *Pad) Paint() {
	p.paint(p.TreeThemePaletteColor("bg"))
}
func (p *Pad) paint(c1 color.Color) {
	b := p.Bounds
	// top
	u := b
	u.Max.Y = u.Min.Y + p.Top
	u = u.Intersect(b)
	imageutil.FillRectangle(p.ctx.Image(), &u, c1)
	// bottom
	u = b
	u.Min.Y = u.Max.Y - p.Bottom
	u = u.Intersect(b)
	imageutil.FillRectangle(p.ctx.Image(), &u, c1)
	// right
	u = b
	u.Min.X = u.Max.X - p.Right
	u = u.Intersect(b)
	imageutil.FillRectangle(p.ctx.Image(), &u, c1)
	// left
	u = b
	u.Max.X = u.Min.X + p.Left
	u = u.Intersect(b)
	imageutil.FillRectangle(p.ctx.Image(), &u, c1)
}
