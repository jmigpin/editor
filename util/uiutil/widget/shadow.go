package widget

import (
	"image"

	"github.com/jmigpin/editor/util/imageutil"
)

type Shadow struct {
	EmbedNode
	ctx         Context
	MaxShade    float64
	Top, Bottom int
}

func NewShadow(ctx Context, child Node) *Shadow {
	s := &Shadow{ctx: ctx, MaxShade: 0.30}
	s.Append(child)
	return s
}
func (s *Shadow) OnMarkChildNeedsPaint(child Node, r *image.Rectangle) {
	// top
	if s.Top > 0 {
		r2 := s.Bounds
		r2.Max.Y = r2.Min.Y + s.Top
		if r2.Overlaps(*r) {
			s.MarkNeedsPaint()
		}
	}
	// bottom
	if s.Bottom > 0 {
		r2 := s.Bounds
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
		h = MaxPoint(h, image.Point{0, 0})
		m := s.EmbedNode.Measure(h)
		m.Y += s.Bottom
		m = MinPoint(m, hint)
		return m
	}
	return s.EmbedNode.Measure(hint)
}

func (s *Shadow) CalcChildsBounds() {
	if s.Bottom > 0 {
		b := s.Bounds
		b.Max.Y -= s.Bottom
		b.Max = MaxPoint(b.Max, image.Point{0, 0})
		child := s.FirstChildInAll()
		child.Embed().Bounds = b
		child.CalcChildsBounds()
		return
	}
	s.EmbedNode.CalcChildsBounds()
}

func (s *Shadow) PaintChilds() {
	// childs are painted first at the top of Paint()
}
func (s *Shadow) Paint() {
	s.EmbedNode.PaintChilds()
	if s.Top > 0 {
		b := s.Bounds
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
		b := s.Bounds
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
