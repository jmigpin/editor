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
	hint.X -= p.Right + p.Left
	hint.Y -= p.Top + p.Bottom
	m := p.FirstChild().Measure(hint)
	m.X += p.Right + p.Left
	m.Y += p.Top + p.Bottom
	return m
}
func (p *Pad) CalcChildsBounds() {
	u := p.Bounds()
	u.Min = u.Min.Add(image.Point{p.Left, p.Top})
	u.Max = u.Max.Sub(image.Point{p.Right, p.Bottom})
	child := p.FirstChild()
	child.SetBounds(&u)
	child.CalcChildsBounds()
}
func (p *Pad) Paint() {
	if p.Color == nil {
		return
	}
	// top
	u := p.Bounds()
	u.Max.Y = u.Min.Y + p.Top
	imageutil.FillRectangle(p.ctx.Image(), &u, *p.Color)
	// bottom
	u = p.Bounds()
	u.Min.Y = u.Max.Y - p.Bottom
	imageutil.FillRectangle(p.ctx.Image(), &u, *p.Color)
	// right
	u = p.Bounds()
	u.Min.X = u.Max.X - p.Right
	imageutil.FillRectangle(p.ctx.Image(), &u, *p.Color)
	// left
	u = p.Bounds()
	u.Max.X = u.Min.X + p.Left
	imageutil.FillRectangle(p.ctx.Image(), &u, *p.Color)
}
