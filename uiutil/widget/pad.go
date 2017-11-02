package widget

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/imageutil"
)

// Can be used as border. If no color is set, it won't paint.
type Pad struct {
	ShellEmbedNode
	Top, Right, Bottom, Left int
	Color                    *color.Color
	ctx                      Context
}

func (p *Pad) Init(ctx Context, child Node) {
	*p = Pad{}
	p.SetWrapper(p)
	p.ctx = ctx
	p.Append(child)
}
func (p *Pad) Set(v int) {
	p.Top = v
	p.Right = v
	p.Bottom = v
	p.Left = v
}
func (p *Pad) Measure(hint image.Point) image.Point {
	h := hint
	h.X -= p.Right + p.Left
	h.Y -= p.Top + p.Bottom
	if h.X < 0 {
		h.X = 0
	}
	if h.Y < 0 {
		h.Y = 0
	}
	m := p.ShellEmbedNode.Measure(h)
	m.X += p.Right + p.Left
	m.Y += p.Top + p.Bottom
	if m.X > hint.X {
		m.X = hint.X
	}
	if m.Y > hint.Y {
		m.Y = hint.Y
	}
	return m
}
func (p *Pad) CalcChildsBounds() {
	b := p.Bounds()
	u := b
	u.Min = u.Min.Add(image.Point{p.Left, p.Top})
	u.Max = u.Max.Sub(image.Point{p.Right, p.Bottom})
	u = u.Intersect(b)
	child := p.FirstChild()
	child.SetBounds(&u)
	child.CalcChildsBounds()
}
func (p *Pad) Paint() {
	if p.Color == nil {
		return
	}
	b := p.Bounds()
	// top
	u := b
	u.Max.Y = u.Min.Y + p.Top
	u = u.Intersect(b)
	imageutil.FillRectangle(p.ctx.Image(), &u, *p.Color)
	// bottom
	u = b
	u.Min.Y = u.Max.Y - p.Bottom
	u = u.Intersect(b)
	imageutil.FillRectangle(p.ctx.Image(), &u, *p.Color)
	// right
	u = b
	u.Min.X = u.Max.X - p.Right
	u = u.Intersect(b)
	imageutil.FillRectangle(p.ctx.Image(), &u, *p.Color)
	// left
	u = b
	u.Max.X = u.Min.X + p.Left
	u = u.Intersect(b)
	imageutil.FillRectangle(p.ctx.Image(), &u, *p.Color)
}
