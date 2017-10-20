package widget

import "image"

type Pad struct {
	ShellEmbedNode
	Top, Right, Bottom, Left int
}

func (p *Pad) Init(n Node) {
	AppendChilds(p, n)
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
	p.FirstChild().SetBounds(&u)
	p.FirstChild().CalcChildsBounds()
}
