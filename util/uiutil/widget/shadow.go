package widget

import (
	"image"

	"github.com/jmigpin/editor/util/imageutil"
)

type TopShadow struct {
	ENode
	Height  int
	MaxDiff float64
	ctx     ImageContext
}

func NewTopShadow(ctx ImageContext, content Node) *TopShadow {
	s := &TopShadow{ctx: ctx, MaxDiff: 0.30, Height: 10}
	s.Append(content)
	return s
}

func (s *TopShadow) OnChildMarked(child Node, newMarks Marks) {
	if newMarks.HasAny(MarkNeedsLayout) {
		s.MarkNeedsLayout()
	}
	if newMarks.HasAny(MarkNeedsPaint | MarkChildNeedsPaint) {
		s.MarkNeedsPaint()
	}
}

func (s *TopShadow) ChildsPaintTree() {
	// childs are painted first at the top of Paint()
}
func (s *TopShadow) Paint() {
	s.ENode.ChildsPaintTree()

	r := s.Bounds
	r.Max.Y = r.Min.Y + s.Height
	r = r.Intersect(s.Bounds)

	imageutil.PaintShadow(s.ctx.Image(), r, s.Height, s.MaxDiff)
}

//----------

type BottomShadow struct {
	*BoxLayout
	Height  int
	MaxDiff float64
	ctx     ImageContext
	content Node
}

func NewBottomShadow(ctx ImageContext, content Node) *BottomShadow {
	s := &BottomShadow{
		ctx: ctx, MaxDiff: 0.30, Height: 10, content: content,
	}

	s.BoxLayout = NewBoxLayout()
	s.YAxis = true

	bsp := &BottomShadowPart{bs: s}

	s.Append(content, bsp)
	s.SetChildFlex(content, false, false)
	s.SetChildFill(bsp, true, false)

	return s
}

//----------

type BottomShadowPart struct {
	ENode
	bs *BottomShadow
}

func (s *BottomShadowPart) Measure(hint image.Point) image.Point {
	w := image.Point{0, s.bs.Height}
	w = imageutil.MinPoint(w, hint)
	return w
}
func (s *BottomShadowPart) Paint() {
	imageutil.PaintShadow(s.bs.ctx.Image(), s.Bounds, s.bs.Height, s.bs.MaxDiff)
}

//----------
