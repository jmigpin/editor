package widget

import (
	"image"
	"image/color"
)

type Border struct {
	EmbedNode
	ui                       UIer
	Top, Right, Bottom, Left int
	Color                    color.Color
}

func NewBorder(ui UIer, n Node) *Border {
	b := &Border{ui: ui}
	AppendChilds(b, n)
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
	return b.FirstChild().Measure(hint)
}
func (b *Border) CalcChildsBounds() {
	u := b.Bounds()
	u.Max = u.Max.Sub(image.Point{b.Right, b.Bottom})
	u.Min = u.Min.Add(image.Point{b.Left, b.Top})
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
