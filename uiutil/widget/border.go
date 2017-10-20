package widget

import (
	"image"
	"image/color"
)

type Border struct {
	ShellEmbedNode
	ui                       UIer
	Top, Right, Bottom, Left int
	Color                    color.Color
}

func NewBorder(ui UIer, n Node) *Border {
	var b Border
	b.Init(ui, n)
	return &b
}
func (b *Border) Init(ui UIer, n Node) {
	b.ui = ui
	AppendChilds(b, n)
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
	b.ui.FillRectangle(&u, b.Color)
	// bottom
	u = b.Bounds()
	u.Min.Y = u.Max.Y - b.Bottom
	b.ui.FillRectangle(&u, b.Color)
	// right
	u = b.Bounds()
	u.Min.X = u.Max.X - b.Right
	b.ui.FillRectangle(&u, b.Color)
	// left
	u = b.Bounds()
	u.Max.X = u.Min.X + b.Left
	b.ui.FillRectangle(&u, b.Color)

}
