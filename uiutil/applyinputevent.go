package uiutil

import (
	"image"
	"time"

	"github.com/jmigpin/editor/uiutil/event"
	"github.com/jmigpin/editor/uiutil/widget"
)

// Childs bounds are assumed to be within parents bounds.
// MouseEnter events run on the parents first, while MouseLeave events run on the childs first.
//
// Events order:
// MouseEnter (if not dragging)
// MouseDown
// MouseMove -> MouseDragStart/MouseDragMove
// MouseUp -> MouseDragEnd -> MouseClick -> MouseDoubleClick -> MouseTripleClick -> MouseEnter (after drag end)
// MouseLeave (if not dragging)

func ApplyInputEventInBounds(node widget.Node, ev interface{}, p image.Point) {
	aie.apply(node, ev, p)
}
func InputEventWarpedPointUntilMouseMove(p image.Point) {
	aie.warpedPoint.on = true
	aie.warpedPoint.p = p
}

var aie ApplyInputEvent

type ApplyInputEvent struct {
	drag struct {
		detect bool
		on     bool
		start  bool
		button event.MouseButton
		point  image.Point
		node   widget.Node
	}
	mclick      map[event.MouseButton]*MultipleClickData
	warpedPoint struct {
		on bool
		p  image.Point
	}
}

func (aie *ApplyInputEvent) apply(node widget.Node, ev interface{}, p image.Point) {
	// warped point (override until move) - improves rapid wheel movement position after warping the pointer
	if _, ok := ev.(*event.MouseMove); ok {
		aie.warpedPoint.on = false
	} else if aie.warpedPoint.on {
		p = aie.warpedPoint.p
	}

	dragWasOn := aie.drag.on

	if !aie.drag.on && !aie.drag.detect {
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
		_ = aie.applyInbound(node, ev, p)
	}
	if !aie.drag.on && !aie.drag.detect {
		// catch structural changes in this cycle by running these int the end

		_ = aie.mouseLeave(node, p)

		if dragWasOn {
			// run after mouse leave
			_ = aie.mouseEnter(node, p)
		}
	}
}

func (aie *ApplyInputEvent) mouseEnter(node widget.Node, p image.Point) bool {
	return aie.visitDepthLast(node, p, func(n widget.Node) bool {
		h := false
		if !n.Marks().PointerInside() {
			n.Marks().SetPointerInside(true)
			h = n.OnInputEvent(&event.MouseEnter{}, p) || h
		}
		return h
	})
}

func (aie *ApplyInputEvent) mouseLeave(node widget.Node, p image.Point) bool {
	if !node.Marks().PointerInside() {
		return false
	}
	h := false
	for c := node.LastChild(); c != nil; c = c.Prev() {
		h = aie.mouseLeave(c, p) || h
	}
	if !p.In(node.Bounds()) {
		node.Marks().SetPointerInside(false)
		h = node.OnInputEvent(&event.MouseLeave{}, p) || h
	}
	return h
}

func (aie *ApplyInputEvent) mouseDown(node widget.Node, ev *event.MouseDown, p image.Point) bool {
	return aie.visitDepthFirst(node, p, func(n widget.Node) bool {
		h := n.OnInputEvent(ev, p)

		// deepest node found
		if !aie.drag.detect && !aie.drag.on && !n.Marks().NotDraggable() {
			aie.drag.detect = true
			aie.drag.button = ev.Button
			aie.drag.point = p
			aie.drag.node = n
		}

		return h
	})
}

func (aie *ApplyInputEvent) mouseDragStartMove(ev *event.MouseMove, p image.Point) bool {
	if aie.drag.detect {
		// if it goes outside the margin, it's a drag
		if !aie.detectMove(aie.drag.point, p) {
			// still inside, try to detect again later
			return false
		}

		// detected, drag is on
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

	cleanup := false
	if aie.drag.on && ev.Button == aie.drag.button {
		cleanup = true
		ev2 := &event.MouseDragEnd{p, ev.Button, ev.Modifiers}
		h = aie.drag.node.OnInputEvent(ev2, p) || h
	}

	if aie.drag.detect || cleanup {
		aie.drag.detect = false
		aie.drag.on = false
		aie.drag.start = false
		aie.drag.node = nil
	}
	return h
}

func (aie *ApplyInputEvent) applyInbound(node widget.Node, ev interface{}, p image.Point) bool {
	return aie.visitDepthFirst(node, p, func(n widget.Node) bool {
		return n.OnInputEvent(ev, p)
	})
}

func (aie *ApplyInputEvent) visitDepthFirst(node widget.Node, p image.Point, fn func(widget.Node) bool) bool {
	if p.In(node.Bounds()) {
		return aie.visitDepthFirst2(node, p, fn)
	}
	return false
}
func (aie *ApplyInputEvent) visitDepthFirst2(node widget.Node, p image.Point, fn func(widget.Node) bool) bool {
	h := false
	// later childs could be drawn over previous ones, run loop backwards
	for c := node.LastChild(); c != nil; c = c.Prev() {
		if p.In(c.Bounds()) {
			h = aie.visitDepthFirst2(c, p, fn) || h
			break
		}
	}
	return fn(node) || h
}

func (aie *ApplyInputEvent) visitDepthLast(node widget.Node, p image.Point, fn func(widget.Node) bool) bool {
	if p.In(node.Bounds()) {
		return aie.visitDepthLast2(node, p, fn)
	}
	return false
}
func (aie *ApplyInputEvent) visitDepthLast2(node widget.Node, p image.Point, fn func(widget.Node) bool) bool {
	h := fn(node)
	// later childs could be drawn over previous ones, run loop backwards
	for c := node.LastChild(); c != nil; c = c.Prev() {
		if p.In(c.Bounds()) {
			h = aie.visitDepthLast2(c, p, fn) || h
			break
		}
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
