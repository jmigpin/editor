package widget

import (
	"image"
)

// Start percent sets the child left X bound to the percent of the size.
// This ensures a change in the percentage of a middle child doesn't affect the bounds of the other childs (ex: like causing a small adjustment when resizing).
type StartPercentLayout struct {
	ENode
	YAxis            bool
	MinimumChildSize int

	minp float64
	spm  map[Node]float64 // start percent map: between 0 and 1
}

func NewStartPercentLayout() *StartPercentLayout {
	spl := &StartPercentLayout{
		spm: make(map[Node]float64),
	}

	// ensure append uses at least this insertbefore implementation.
	spl.Wrapper = spl

	return spl
}

//----------

func (spl *StartPercentLayout) Measure(hint image.Point) image.Point {
	return hint
}

func (spl *StartPercentLayout) Layout() {
	// translate axis
	xya := XYAxis{spl.YAxis}
	abounds := xya.Rectangle(&spl.Bounds)

	// update minimum percentage
	spl.minp = float64(spl.MinimumChildSize) / float64(abounds.Size().X)

	// set sizes
	dxf := float64(abounds.Dx())

	spl.IterateWrappers2(func(child Node) {
		sp := spl.sp(child)
		ep := spl.ep(child)

		var r image.Rectangle
		cxs := abounds.Min.X + int(sp*dxf)
		cxe := abounds.Min.X + int(ep*dxf)

		// set bounds
		r.Min = image.Point{cxs, abounds.Min.Y}
		r.Max = image.Point{cxe, abounds.Max.Y}

		// fix last child for rounding errors
		if child == spl.LastChildWrapper() {
			r.Max.X = abounds.Max.X
		}

		// translate axis
		r2 := xya.Rectangle(&r)

		r3 := r2.Intersect(spl.Bounds)
		child.Embed().Bounds = r3
	})
}

//----------

func (spl *StartPercentLayout) InsertBefore(n Node, mark *EmbedNode) {
	if mark != nil && spl.sp(mark.Wrapper) == 0.0 {
		// insert after mark
		spl.ENode.InsertBefore(n, mark.NextSibling())
	} else {
		spl.ENode.InsertBefore(n, mark)
	}

	ne := n.Embed()
	if ne.PrevSiblingWrapper() != nil {
		// share with previous sibling
		s, e := spl.sp(ne.PrevSiblingWrapper()), spl.ep(n)
		spl.setsp(n, s+(e-s)/2)
	} else {
		// first child
		spl.setsp(n, 0)
	}
}

func (spl *StartPercentLayout) Remove(n Node) {
	delete(spl.spm, n)
	spl.ENode.Remove(n)
}

//----------

// start percent
func (spl *StartPercentLayout) sp(n Node) float64 {
	return spl.spm[n]
}

// end percent
func (spl *StartPercentLayout) ep(n Node) float64 {
	ns := n.Embed().NextSiblingWrapper()
	if ns != nil {
		return spl.sp(ns)
	}
	return 1.0
}

// start percent of previous
func (spl *StartPercentLayout) spPrev(n Node) float64 {
	ps := n.Embed().PrevSiblingWrapper()
	if ps != nil {
		return spl.sp(ps)
	}
	return 0.0
}

// set start percent
func (spl *StartPercentLayout) setsp(n Node, v float64) {
	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}
	spl.spm[n] = v

	spl.MarkNeedsLayout()
}

//----------

func (spl *StartPercentLayout) size(n Node) float64 {
	return spl.ep(n) - spl.sp(n)
}
func (spl *StartPercentLayout) prevSize(n Node) float64 {
	return spl.sp(n) - spl.spPrev(n)
}

//----------

func (spl *StartPercentLayout) Resize(node Node, percent float64) {
	spl.resize(node, percent)
}

func (spl *StartPercentLayout) resize(node Node, percent float64) {
	s, e := spl.spPrev(node), spl.ep(node)

	// add margins
	s2, e2 := s, e
	if node.Embed() != spl.FirstChild() {
		s2 += spl.minp
	}
	e2 -= spl.minp

	// squash in the middle
	if node.Embed() != spl.FirstChild() && s2 > e2 {
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

//----------

func (spl *StartPercentLayout) ResizeWithMove(node Node, percent float64) {
	minDir := true
	sp := spl.sp(node)
	if percent >= sp {
		minDir = false
	}
	_ = spl.resizeWithMove(node, percent, minDir, true)
}

func (spl *StartPercentLayout) resizeWithMove(node Node, percent float64, minDir bool, pusher bool) bool {
	// preemptively move node
	moved := false
	if minDir {
		for n := node.Embed().PrevSiblingWrapper(); n != nil; n = n.Embed().PrevSiblingWrapper() {
			s := spl.spPrev(n)
			e := spl.sp(n)
			if s < percent && percent < e {
				// directly use EmbedNode remove/insertbefore
				spl.ENode.Remove(node)
				spl.ENode.InsertBefore(node, n.Embed())
				moved = true
				break
			}
		}
	} else {
		for n := node.Embed().NextSiblingWrapper(); n != nil; n = n.Embed().NextSiblingWrapper() {
			s := spl.sp(n)
			e := spl.ep(n)
			if s < percent && percent < e {
				// directly use EmbedNode remove/insertbefore
				spl.ENode.Remove(node)
				spl.ENode.InsertBefore(node, n.Embed().NextSibling())
				moved = true
				break
			}
		}
	}

	spl.resize(node, percent)

	return moved
}

//----------

func (spl *StartPercentLayout) SetPercentWithPush(node Node, percentPos float64) {
	sp := spl.sp(node)
	if percentPos > sp {
		percent := percentPos - sp
		_ = spl.incStartBy(node, percent)
	} else {
		percent := sp - percentPos
		_ = spl.decStartBy(node, percent)
	}
}

//----------

func (spl *StartPercentLayout) SetSizePercentWithPush(node Node, sizePercent float64) {
	size := spl.size(node)
	if size < sizePercent {
		d := sizePercent - size
		// push next siblings
		d -= spl.incStartBy(node.Embed().NextSiblingWrapper(), d)
		// push prev siblings
		_ = spl.decStartBy(node, d)
	}
}

//----------

func (spl *StartPercentLayout) incStartBy(node Node, percent float64) float64 {
	if percent < 0.00001 {
		return 0.0
	}
	if node == nil {
		return 0.0
	}
	size := spl.size(node)
	if size-percent >= spl.minp {
		spl.setsp(node, spl.sp(node)+percent)
		return percent
	}

	// at this node
	w := size - spl.minp
	if w < 0 {
		w = 0
	}
	spl.setsp(node, spl.sp(node)+w)

	// at siblings
	w2 := spl.incStartBy(node.Embed().NextSiblingWrapper(), percent-w)
	spl.setsp(node, spl.sp(node)+w2)

	return w + w2
}

func (spl *StartPercentLayout) decStartBy(node Node, percent float64) float64 {
	if percent < 0.00001 {
		return 0.0
	}
	if node == nil {
		return 0.0
	}

	minp := spl.minp
	if node.Embed() == spl.FirstChild() {
		minp = 0.0
	}

	size := spl.prevSize(node)
	if size-percent >= minp {
		spl.setsp(node, spl.sp(node)-percent)
		return percent
	}

	// at this node
	w := size - minp
	if w < 0 {
		w = 0
	}
	spl.setsp(node, spl.sp(node)-w)

	// at siblings
	w2 := spl.decStartBy(node.Embed().PrevSiblingWrapper(), percent-w)
	spl.setsp(node, spl.sp(node)-w2)

	return w + w2
}

//----------

func (spl *StartPercentLayout) MaximizeNode(node Node) {
	n := spl.ChildsLen()
	sp := 0.0
	i := 0
	spl.IterateWrappers2(func(c Node) {
		spl.setsp(c, sp)
		if c == node {
			sp = 1.0 - spl.minp*float64(n-(i+1))
		} else {
			sp += spl.minp
		}
		i++
	})
}

//----------

// Used for encoding/decoding only. (Ex: sessions)
func (spl *StartPercentLayout) SetRawStartPercent(child Node, v float64) {
	spl.setsp(child, v)
}

// Used for encoding/decoding only. (Ex: sessions)
func (spl *StartPercentLayout) RawStartPercent(child Node) float64 {
	return spl.sp(child)
}
