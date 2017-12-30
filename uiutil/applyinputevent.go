package uiutil

import (
	"image"
	"time"

	"github.com/jmigpin/editor/uiutil/event"
	"github.com/jmigpin/editor/uiutil/widget"
)

// MouseEnter events run on the parents first, while MouseLeave events run on the childs first.

// Events order:
// MouseLeave/MouseEnter
// MouseDown
// MouseMove -> MouseDragStart -> MouseDragMove
// MouseUp -> MouseDragEnd -> MouseClick -> MouseDoubleClick -> MouseTripleClick

var AIE ApplyInputEvent

type ApplyInputEvent struct {
	drag        AIEDrag
	mclick      map[event.MouseButton]*MultipleClickData
	warpedPoint struct {
		on bool
		p  image.Point
	}
}

type AIEDrag struct {
	pressing bool
	detect   bool
	on       bool
	start    bool
	button   event.MouseButton
	point    image.Point
	node     widget.Node
}

func (aie *ApplyInputEvent) SetWarpedPointUntilMouseMove(p image.Point) {
	aie.warpedPoint.on = true
	aie.warpedPoint.p = p
}

func (aie *ApplyInputEvent) getWarpedPoint(ev interface{}, p image.Point) image.Point {
	// warped point overrides the point until move - improves rapid wheel movement position after warping the pointer
	if _, ok := ev.(*event.MouseMove); ok {
		aie.warpedPoint.on = false
	} else if aie.warpedPoint.on {
		return aie.warpedPoint.p
	}
	return p
}

func (aie *ApplyInputEvent) Apply(ctx widget.CursorContext, node widget.Node, ev interface{}, p image.Point) {
	p = aie.getWarpedPoint(ev, p)

	dragWasOn := aie.drag.on

	if !aie.drag.pressing {
		_ = aie.mouseLeave(node, p)
		_ = aie.mouseEnter(node, p)
	}

	switch evt := ev.(type) {
	case *event.MouseDown:
		_ = aie.mouseDown(node, evt, p) // sets drag.detect
		aie.multipleClickMouseDown(node, evt, p)
	case *event.MouseMove:
		_ = aie.applyInbound(node, evt, p)
		_ = aie.mouseDragStartMove(evt, p) // sets drag.on
	case *event.MouseUp:
		_ = aie.applyInbound(node, evt, p)
		_ = aie.mouseDragEnd(evt, p) // clears drag.on
		if !dragWasOn {
			_ = aie.multipleClickMouseUp(node, evt, p)
		}
	default:
		// ex: event.KeyDown
		_ = aie.applyInbound(node, evt, p)
	}

	if !aie.drag.on {
		widget.SetTreeCursor(ctx, node, p)
	}

	//// catch structural changes in this cycle by running these int the end
	//// ex: allows visual update of a widget that closed and the mouse didn't move
	//if !aie.drag.pressing {
	//	_ = aie.mouseLeave(node, p)
	//	//if dragWasOn {
	//	// run after mouse leave
	//	_ = aie.mouseEnter(node, p)
	//	//}
	//}
}

func (aie *ApplyInputEvent) applyInbound(node widget.Node, ev interface{}, p image.Point) bool {
	return aie.visitDepthFirst(node, p, func(n widget.Node) bool {
		return n.OnInputEvent(ev, p)
	})
}

func (aie *ApplyInputEvent) mouseEnter(node widget.Node, p image.Point) bool {
	return aie.visitDepthLast(node, p, func(n widget.Node) bool {
		h := false
		if !n.Embed().PointerInside() {
			n.Embed().SetPointerInside(true)
			h = n.OnInputEvent(&event.MouseEnter{}, p) || h
		}
		return false // never early exit
	})
}

func (aie *ApplyInputEvent) mouseLeave(node widget.Node, p image.Point) bool {
	if !node.Embed().PointerInside() {
		return false
	}

	// execute on childs
	h := false
	node.Embed().IterChildsReverse(func(c widget.Node) {
		h = aie.mouseLeave(c, p) || h
	})

	// execute on node
	if !p.In(node.Embed().Bounds) {
		node.Embed().SetPointerInside(false)
		h = node.OnInputEvent(&event.MouseLeave{}, p) || h
	}

	return false // never early exit
}

func (aie *ApplyInputEvent) mouseDown(node widget.Node, ev *event.MouseDown, p image.Point) bool {
	return aie.visitDepthFirst(node, p, func(n widget.Node) bool {
		h := n.OnInputEvent(ev, p)

		// deepest node found
		if !aie.drag.pressing && !n.Embed().NotDraggable() {
			aie.drag.pressing = true
			aie.drag.button = ev.Button
			aie.drag.detect = true
			aie.drag.point = p
			aie.drag.node = n
		}

		return h
	})
}

func (aie *ApplyInputEvent) mouseDragStartMove(ev *event.MouseMove, p image.Point) bool {
	if aie.drag.detect {
		// still haven't move enough, try to detect again later
		if !aie.detectMove(aie.drag.point, p) {
			return false
		}
		// dragging
		aie.drag.on = true
		aie.drag.detect = false
	}

	h := false
	if aie.drag.on {
		if !aie.drag.start {
			aie.drag.start = true
			ev2 := &event.MouseDragStart{p, aie.drag.button, ev.Modifiers}
			h = aie.drag.node.OnInputEvent(ev2, p) || h
		} else {
			ev2 := &event.MouseDragMove{p, ev.Buttons, ev.Modifiers}
			h = aie.drag.node.OnInputEvent(ev2, p) || h
		}
	}
	return h
}
func (aie *ApplyInputEvent) mouseDragEnd(ev *event.MouseUp, p image.Point) bool {
	h := false
	if aie.drag.pressing && ev.Button == aie.drag.button {
		if aie.drag.on {
			ev2 := &event.MouseDragEnd{p, ev.Button, ev.Modifiers}
			h = aie.drag.node.OnInputEvent(ev2, p)
		}
		// cleanup
		aie.drag = AIEDrag{}
	}
	return h
}

func (aie *ApplyInputEvent) visitDepthFirst(node widget.Node, p image.Point, fn func(widget.Node) bool) bool {
	if !p.In(node.Embed().Bounds) {
		return false
	}

	// execute on childs
	h := false
	// later childs are drawn over previous ones, run loop backwards
	node.Embed().IterChildsReverseStop(func(c widget.Node) bool {
		h = aie.visitDepthFirst(c, p, fn)
		// early exit if handled
		if h {
			return false
		}
		return true
	})

	// execute on node
	if !h {
		h = fn(node)
	}

	return h
}
func (aie *ApplyInputEvent) visitDepthLast(node widget.Node, p image.Point, fn func(widget.Node) bool) bool {
	if !p.In(node.Embed().Bounds) {
		return false
	}

	// execute on node
	h := fn(node)

	// execute on childs
	if !h {
		// later childs are drawn over previous ones, run loop backwards
		node.Embed().IterChildsReverseStop(func(c widget.Node) bool {
			h = aie.visitDepthLast(c, p, fn)
			// early exit if handled
			if h {
				return false
			}
			return true
		})
	}

	return h
}

func (aie *ApplyInputEvent) multipleClickLazyInit() {
	if aie.mclick == nil {
		aie.mclick = make(map[event.MouseButton]*MultipleClickData)
	}
}

func (aie *ApplyInputEvent) multipleClickMouseDown(node widget.Node, ev *event.MouseDown, p image.Point) {
	aie.multipleClickLazyInit()
	mc, ok := aie.mclick[ev.Button]
	if !ok {
		mc = &MultipleClickData{}
		aie.mclick[ev.Button] = mc
	}
	mc.PrevPoint = mc.Point
	mc.Point = p
}
func (aie *ApplyInputEvent) multipleClickMouseUp(node widget.Node, ev *event.MouseUp, p image.Point) bool {
	aie.multipleClickLazyInit()
	mc, ok := aie.mclick[ev.Button]
	if !ok {
		return false
	}

	// update time
	t0 := mc.T
	mc.T = time.Now()

	// must be clicked within a margin
	if aie.detectMove(mc.Point, p) {
		mc.Action = 0
		return false
	}

	// if it takes too much time, it gets back to single click
	d := mc.T.Sub(t0)
	if d > 400*time.Millisecond {
		mc.Action = 0
	} else {
		if aie.detectMove(mc.PrevPoint, p) {
			mc.Action = 0
		} else {
			mc.Action = (mc.Action + 1) % 3 // single, double, triple
		}
	}

	h := false

	// always run a click
	ev2 := &event.MouseClick{p, ev.Button, ev.Modifiers}
	u := aie.applyInbound(node, ev2, p)
	h = h || u

	switch mc.Action {
	case 1:
		ev2 := &event.MouseDoubleClick{p, ev.Button, ev.Modifiers}
		u = aie.applyInbound(node, ev2, p)
		h = h || u
	case 2:
		ev2 := &event.MouseTripleClick{p, ev.Button, ev.Modifiers}
		u = aie.applyInbound(node, ev2, p)
		h = h || u
	}
	return h
}

func (aie *ApplyInputEvent) detectMove(p0, p1 image.Point) bool {
	sidePad := image.Point{3, 3}
	var r image.Rectangle
	r.Min = p0.Sub(sidePad)
	r.Max = p0.Add(sidePad)
	return !p1.In(r)
}

type MultipleClickData struct {
	T         time.Time
	Point     image.Point
	PrevPoint image.Point
	Action    int // single, double, triple
}
