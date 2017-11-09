package widget

import (
	"image"

	"github.com/jmigpin/editor/imageutil"
)

type Shadow struct {
	ShellEmbedNode
	ctx         Context
	MaxShade    float64
	Top, Bottom int
}

func (s *Shadow) Init(ctx Context, child Node) {
	*s = Shadow{ctx: ctx, MaxShade: 0.30}
	s.SetWrapper(s)
	s.Append(child)
}
func (s *Shadow) OnMarkChildNeedsPaint(child Node, r *image.Rectangle) {
	// top
	if s.Top > 0 {
		r2 := s.Bounds()
		r2.Max.Y = r2.Min.Y + s.Top
		if r2.Overlaps(*r) {
			s.MarkNeedsPaint()
		}
	}
	// bottom
	if s.Bottom > 0 {
		r2 := s.Bounds()
		r2.Min.Y = r2.Max.Y - s.Bottom
		if r2.Overlaps(*r) {
			s.MarkNeedsPaint()
		}
	}
}
func (s *Shadow) Measure(hint image.Point) image.Point {
	if s.Bottom > 0 {
		h := hint
		h.Y -= s.Bottom
		if h.Y < 0 {
			h.Y = 0
		}
		m := s.ShellEmbedNode.Measure(h)
		m.Y += s.Bottom
		if m.Y > hint.Y {
			m.Y = hint.Y
		}
		return m
	}
	return s.ShellEmbedNode.Measure(hint)
}
func (s *Shadow) CalcChildsBounds() {
	if s.Bottom > 0 {
		b := s.Bounds()
		b.Max.Y -= s.Bottom
		if b.Max.Y < 0 {
			b.Max.Y = 0
		}
		child := s.FirstChild()
		child.SetBounds(&b)
		child.CalcChildsBounds()
		return
	}
	s.ShellEmbedNode.CalcChildsBounds()
}
func (s *Shadow) PaintChilds() {
	// childs are painted first - at the top of Paint()
}
func (s *Shadow) Paint() {
	s.ShellEmbedNode.PaintChilds()
	if s.Top > 0 {
		b := s.Bounds()
		j := 0.0
		img := s.ctx.Image()
		maxY := b.Min.Y + s.Top
		if maxY > b.Max.Y {
			maxY = b.Max.Y
		}
		for y := b.Min.Y; y < maxY; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				at := img.At(x, y)
				c2 := imageutil.Shade(at, s.MaxShade-j)
				img.Set(x, y, c2)
			}
			j += s.MaxShade / float64(s.Top)
		}
	}
	if s.Bottom > 0 {
		b := s.Bounds()
		j := 0.0
		img := s.ctx.Image()
		for y := b.Max.Y - s.Bottom; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				at := img.At(x, y)
				c2 := imageutil.Shade(at, s.MaxShade-j)
				img.Set(x, y, c2)
			}
			j += s.MaxShade / float64(s.Bottom)
		}
	}
}
