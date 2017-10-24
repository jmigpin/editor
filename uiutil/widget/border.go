package widget

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/imageutil"
)

type Border struct {
	ShellEmbedNode
	Top, Right, Bottom, Left int
	Color                    color.Color
	ctx                      Context
}

func NewBorder(ctx Context, n Node) *Border {
	b := &Border{}
	b.SetWrapper(b)
	b.ctx = ctx
	b.Append(n)
	return b
}
func (b *Border) Set(v int) {
	b.Top = v
	b.Right = v
	b.Bottom = v
	b.Left = v
}
func (b *Border) Measure(hint image.Point) image.Point {
	hint.X -= b.Right + b.Left
	hint.Y -= b.Top + b.Bottom
	m := b.FirstChild().Measure(hint)
	m.X += b.Right + b.Left
	m.Y += b.Top + b.Bottom
	return m
}
func (b *Border) CalcChildsBounds() {
	u := b.Bounds()
	u.Min = u.Min.Add(image.Point{b.Left, b.Top})
	u.Max = u.Max.Sub(image.Point{b.Right, b.Bottom})
	b.FirstChild().SetBounds(&u)
	b.FirstChild().CalcChildsBounds()
}
func (b *Border) Paint() {
	// top
	u := b.Bounds()
	u.Max.Y = u.Min.Y + b.Top
	imageutil.FillRectangle(b.ctx.Image(), &u, b.Color)
	// bottom
	u = b.Bounds()
	u.Min.Y = u.Max.Y - b.Bottom
	imageutil.FillRectangle(b.ctx.Image(), &u, b.Color)
	// right
	u = b.Bounds()
	u.Min.X = u.Max.X - b.Right
	imageutil.FillRectangle(b.ctx.Image(), &u, b.Color)
	// left
	u = b.Bounds()
	u.Max.X = u.Min.X + b.Left
	imageutil.FillRectangle(b.ctx.Image(), &u, b.Color)
}
