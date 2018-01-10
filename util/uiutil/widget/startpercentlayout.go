package widget

import (
	"image"
	"reflect"
)

// Start percent sets the child left X bound to the percent of the size.
// This ensures a change in the percentage of a middle child doesn't affect the bounds of the other childs (ex: like causing a small adjustment when resizing).

type StartPercentLayout struct {
	EmbedNode
	YAxis            bool
	MinimumChildSize int

	minp float64
	spm  map[Node]float64 // between 0 and 1
}

func NewStartPercentLayout() *StartPercentLayout {
	spl := &StartPercentLayout{
		spm: make(map[Node]float64),
	}

	// ensure append uses this insertbefore implementation.
	spl.wrapper = spl

	return spl
}

func (spl *StartPercentLayout) Measure(hint image.Point) image.Point {
	return hint
}

func (spl *StartPercentLayout) CalcChildsBounds() {
	// translate axis
	xya := XYAxis{spl.YAxis}
	abounds := xya.Rectangle(&spl.Bounds)

	// update minimum percentage
	spl.minp = float64(spl.MinimumChildSize) / float64(abounds.Size().X)

	// set sizes
	dxf := float64(abounds.Dx())
	spl.IterChilds(func(child Node) {
		sp := spl.sp(child)
		ep := spl.ep(child)

		var r image.Rectangle
		cxs := abounds.Min.X + int(sp*dxf)
		cxe := abounds.Min.X + int(ep*dxf)
		r.Min = image.Point{cxs, abounds.Min.Y}
		r.Max = image.Point{cxe, abounds.Max.Y}

		// fix last child for rounding errors
		if child == spl.LastChild() {
			r.Max.X = abounds.Max.X
		}

		// translate axis
		r2 := xya.Rectangle(&r)

		r3 := r2.Intersect(spl.Bounds)
		child.Embed().Bounds = r3
		child.CalcChildsBounds()
	})
}

func (spl *StartPercentLayout) InsertBefore(n, mark Node) {
	if (mark != nil && !reflect.ValueOf(mark).IsNil()) && spl.sp(mark) == 0.0 {
		// insert after mark
		spl.EmbedNode.InsertBefore(n, mark.Embed().Next())
	} else {
		spl.EmbedNode.InsertBefore(n, mark)
	}

	ne := n.Embed()
	if ne.Prev() != nil {
		// share with previous sibling
		s, e := spl.sp(ne.Prev()), spl.ep(n)
		spl.setsp(n, s+(e-s)/2)
	} else {
		// first child
		spl.setsp(n, 0)
	}
}

func (spl *StartPercentLayout) Remove(n Node) {
	delete(spl.spm, n)
	spl.EmbedNode.Remove(n)
}

// start percent
func (spl *StartPercentLayout) sp(n Node) float64 {
	return spl.spm[n]
}

// end percent
func (spl *StartPercentLayout) ep(n Node) float64 {
	ne := n.Embed()
	if ne.Next() != nil {
		return spl.sp(ne.Next())
	}
	return 1.0
}

// start percent of previous
func (spl *StartPercentLayout) spPrev(n Node) float64 {
	ne := n.Embed()
	if ne.Prev() != nil {
		return spl.sp(ne.Prev())
	}
	return 0.0
}

func (spl *StartPercentLayout) setsp(n Node, v float64) {
	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}
	spl.spm[n] = v
}

func (spl *StartPercentLayout) resize(node Node, percent float64) {
	s, e := spl.spPrev(node), spl.ep(node)

	// add margins
	s2, e2 := s, e
	if node != spl.FirstChild() {
		s2 += spl.minp
	}
	e2 -= spl.minp

	// squash in the middle
	if node != spl.FirstChild() && s2 > e2 {
		spl.setsp(node, s+(e-s)/2)
		return
	}

	// minimum
	if percent < s2 {
		spl.setsp(node, s2)
		return
	}
	// maximum
	if percent > e2 {
		spl.setsp(node, e2)
		return
	}

	spl.setsp(node, percent)
}

func (spl *StartPercentLayout) Resize(node Node, percent float64) {
	spl.resize(node, percent)
}

func (spl *StartPercentLayout) resizeWithPush(node Node, percent float64, minDir bool, pusher bool) {
	// preemptively resize neighbour
	ne := node.Embed()
	if minDir {
		if ne.Prev() != nil {
			spl.resizeWithPush(ne.Prev(), percent-spl.minp, minDir, false)
		}
	} else {
		if ne.Next() != nil {
			spl.resizeWithPush(ne.Next(), percent+spl.minp, minDir, false)
		}
	}

	// check if the percent is already satisfied (no need to resize)
	if !pusher {
		sp := spl.sp(node)
		if minDir {
			if sp <= percent {
				return
			}
		} else {
			if sp >= percent {
				return
			}
		}
	}

	// resize to satisfy the requested percent
	spl.resize(node, percent)
}

func (spl *StartPercentLayout) ResizeWithPush(node Node, percent float64) {
	minDir := true
	sp := spl.sp(node)
	if percent >= sp {
		minDir = false
	}
	spl.resizeWithPush(node, percent, minDir, true)
}

func (spl *StartPercentLayout) resizeWithMove(node Node, percent float64, minDir bool, pusher bool) bool {
	// preemptively move node
	moved := false
	if minDir {
		for n := node.Embed().Prev(); n != nil; n = n.Embed().Prev() {
			s := spl.spPrev(n)
			e := spl.sp(n)
			if s < percent && percent < e {
				// directly use EmbedNode remove/insertbefore
				spl.EmbedNode.Remove(node)
				spl.EmbedNode.InsertBefore(node, n)
				moved = true
				break
			}
		}
	} else {
		for n := node.Embed().Next(); n != nil; n = n.Embed().Next() {
			s := spl.sp(n)
			e := spl.ep(n)
			if s < percent && percent < e {
				// directly use EmbedNode remove/insertbefore
				spl.EmbedNode.Remove(node)
				spl.EmbedNode.InsertBefore(node, n.Embed().Next())
				moved = true
				break
			}
		}
	}

	spl.resize(node, percent)

	return moved
}

func (spl *StartPercentLayout) ResizeWithMove(node Node, percent float64) {
	minDir := true
	sp := spl.sp(node)
	if percent >= sp {
		minDir = false
	}
	_ = spl.resizeWithMove(node, percent, minDir, true)
}

func (spl *StartPercentLayout) MaximizeNode(node Node) {
	n := spl.ChildsLen()
	sp := 0.0
	i := 0
	spl.IterChilds(func(c Node) {
		spl.setsp(c, sp)
		if c == node {
			sp = 1.0 - spl.minp*float64(n-(i+1))
		} else {
			sp += spl.minp
		}
		i++
	})
}

// Used for encoding/decoding only. (Ex: sessions)
func (spl *StartPercentLayout) SetRawStartPercent(child Node, v float64) {
	spl.spm[child] = v
}

// Used for encoding/decoding only. (Ex: sessions)
func (spl *StartPercentLayout) RawStartPercent(child Node) float64 {
	return spl.spm[child]
}
