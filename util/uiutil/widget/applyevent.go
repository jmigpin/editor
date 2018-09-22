package widget

import (
	"image"
	"time"

	"github.com/jmigpin/editor/util/uiutil/event"
)

type ApplyEvent struct {
	drag   AEDragState
	mclick map[event.MouseButton]*AEMultipleClick

	cctx CursorContext
}

func NewApplyEvent(cctx CursorContext) *ApplyEvent {
	ae := &ApplyEvent{cctx: cctx}
	ae.mclick = map[event.MouseButton]*AEMultipleClick{}
	return ae
}

//----------

func (ae *ApplyEvent) Apply(node Node, ev interface{}, p image.Point) {
	ae.mouseEnterLeave(node, p)

	switch evt := ev.(type) {
	case *event.MouseDown:
		ae.depthFirstEv(node, evt, p)
		ae.findDragNode(node, evt, p)
		ae.multipleClickMouseDown(node, evt, p)
	case *event.MouseMove:
		ae.depthFirstEv(node, evt, p)
		ae.dragStartMove(evt, p)
	case *event.MouseUp:
		ae.depthFirstEv(node, evt, p)
		ae.dragEnd(evt, p)
		ae.multipleClickMouseUp(node, evt, p)
		// mouseup can cause a ui change, enter/leave needs to run
		ae.mouseEnterLeave(node, p)
	case *event.KeyDown:
		isLatch := event.ComposeDiacritic(&evt.KeySym, &evt.Rune)
		if !isLatch {
			ae.depthFirstEv(node, evt, p)
		}
	default:
		// ex: event.KeyUp
		ae.depthFirstEv(node, evt, p)
	}

	ae.setCursor(node, p)
}

//----------

func (ae *ApplyEvent) setCursor(node Node, p image.Point) {
	var c Cursor
	if ae.drag.pressing {
		c = ae.drag.node.Embed().Cursor
	} else {
		c = ae.treeCursor(node, p)
	}
	ae.cctx.SetCursor(c)
}

func (ae *ApplyEvent) treeCursor(node Node, p image.Point) Cursor {
	ne := node.Embed()
	if !p.In(ne.Bounds) {
		return 0
	}
	var c Cursor
	ne.IterateWrappersReverse(func(child Node) bool {
		c = ae.treeCursor(child, p)
		return c == 0 // continue while no cursor was set
	})
	if c == 0 {
		c = ne.Cursor
	}
	return c
}

//----------

func (ae *ApplyEvent) mouseEnterLeave(node Node, p image.Point) {
	if ae.drag.pressing {
		return
	}
	ae.mouseLeave(node, p) // run leave first, then enter another node (correctness)
	ae.mouseEnter(node, p)
}

//----------

func (ae *ApplyEvent) mouseEnter(node Node, p image.Point) event.Handle {
	ne := node.Embed()

	if !p.In(ne.Bounds) {
		return event.NotHandled
	}

	// execute on childs
	h := event.NotHandled
	// later childs are drawn over previous ones, run loop backwards
	ne.IterateWrappersReverse(func(c Node) bool {
		h = ae.mouseEnter(c, p)
		return h == event.NotHandled // continue while not handled
	})

	// execute on node
	if h == event.NotHandled {
		if !ne.Marks.HasAny(MarkPointerInside) {
			ne.Marks.Add(MarkPointerInside)
			ev2 := &event.MouseEnter{}
			h = ae.runEv(node, ev2, p)
		}
	}

	if ne.Marks.HasAny(MarkInBoundsHandlesEvent) {
		h = event.Handled
	}

	return h
}

//----------

func (ae *ApplyEvent) mouseLeave(node Node, p image.Point) event.Handle {
	ne := node.Embed()

	// execute on childs
	h := event.NotHandled
	// later childs are drawn over previous ones, run loop backwards
	ne.IterateWrappersReverse(func(c Node) bool {
		h = ae.mouseLeave(c, p)
		return h == event.NotHandled // continue while not handled
	})

	// execute on node
	if h == event.NotHandled {
		if ne.Marks.HasAny(MarkPointerInside) && !p.In(ne.Bounds) {
			ne.Marks.Remove(MarkPointerInside)
			ev2 := &event.MouseLeave{}
			h = ae.runEv(node, ev2, p)
		}
	}

	return h
}

//----------

func (ae *ApplyEvent) findDragNode(node Node, ev *event.MouseDown, p image.Point) {
	if ae.drag.pressing {
		return
	}
	ae.findDragNode2(node, ev, p)
}

// Depth first, reverse order.
func (ae *ApplyEvent) findDragNode2(node Node, ev *event.MouseDown, p image.Point) bool {
	if !p.In(node.Embed().Bounds) {
		return false
	}

	// execute on childs
	found := false
	node.Embed().IterateWrappersReverse(func(c Node) bool {
		found = ae.findDragNode2(c, ev, p)
		return !found // continue while not found
	})

	if !found {
		// deepest node
		canDrag := !node.Embed().Marks.HasAny(MarkNotDraggable)
		if canDrag {
			ae.drag.pressing = true
			ae.drag.node = node
			ae.drag.point = p
			ae.drag.button = ev.Button
			return true
		}
	}

	return found
}

//----------

func (ae *ApplyEvent) dragStartMove(ev *event.MouseMove, p image.Point) {
	if !ae.drag.pressing {
		return
	}
	if !ae.drag.on {
		// still haven't move enough, try to detect again later
		if !ae.detectMove(ae.drag.point, p) {
			return
		}
		// dragging
		ae.drag.on = true
	}
	if !ae.drag.start {
		ae.drag.start = true
		ev2 := &event.MouseDragStart{p, ae.drag.button, ev.Mods}
		ae.runEv(ae.drag.node, ev2, p)
	} else {
		ev2 := &event.MouseDragMove{p, ev.Buttons, ev.Mods}
		ae.runEv(ae.drag.node, ev2, p)
	}
}

//----------

func (ae *ApplyEvent) dragEnd(ev *event.MouseUp, p image.Point) {
	if !ae.drag.pressing {
		return
	}
	if ev.Button != ae.drag.button {
		return
	}

	if ae.drag.on {
		ev2 := &event.MouseDragEnd{p, ev.Button, ev.Mods}
		ae.runEv(ae.drag.node, ev2, p)
	}

	ae.drag = AEDragState{}
}

//----------

func (ae *ApplyEvent) depthFirstEv(node Node, ev interface{}, p image.Point) event.Handle {
	if !p.In(node.Embed().Bounds) {
		return event.NotHandled
	}

	// execute on childs
	h := event.NotHandled
	// later childs are drawn over previous ones, run loop backwards
	node.Embed().IterateWrappersReverse(func(c Node) bool {
		h = ae.depthFirstEv(c, ev, p)
		return h == event.NotHandled // continue while not handled
	})

	// execute on node
	if h == event.NotHandled {
		h = ae.runEv(node, ev, p)
	}

	if node.Embed().Marks.HasAny(MarkInBoundsHandlesEvent) {
		h = event.Handled
	}

	return h
}

//----------

func (ae *ApplyEvent) runEv(node Node, ev interface{}, p image.Point) event.Handle {
	return node.OnInputEvent(ev, p)
}

//----------

func (ai *ApplyEvent) detectMove(p0, p1 image.Point) bool {
	sidePad := image.Point{3, 3}
	var r image.Rectangle
	r.Min = p0.Sub(sidePad)
	r.Max = p0.Add(sidePad)
	return !p1.In(r)
}

//----------

func (ae *ApplyEvent) multipleClickMouseDown(node Node, ev *event.MouseDown, p image.Point) {
	mc, ok := ae.mclick[ev.Button]
	if !ok {
		mc = &AEMultipleClick{}
		ae.mclick[ev.Button] = mc
	}
	mc.prevDownPoint = mc.downPoint
	mc.downPoint = p
}

//----------

func (ae *ApplyEvent) multipleClickMouseUp(node Node, ev *event.MouseUp, p image.Point) {
	mc, ok := ae.mclick[ev.Button]
	if !ok {
		return
	}

	// update time
	upTime0 := mc.upTime
	mc.upTime = time.Now()

	// must be clicked within a margin
	if ae.detectMove(mc.downPoint, p) {
		mc.action = 0
		return
	}

	// if it takes too much time, it gets back to single click
	d := mc.upTime.Sub(upTime0)
	if d > 400*time.Millisecond {
		mc.action = 0
	} else {
		if ae.detectMove(mc.prevDownPoint, p) {
			mc.action = 0
		} else {
			// single, double, triple
			mc.action = (mc.action + 1) % 3
		}
	}

	// always run a click
	ev2 := &event.MouseClick{p, ev.Button, ev.Mods}
	ae.depthFirstEv(node, ev2, p)

	switch mc.action {
	case 1:
		ev2 := &event.MouseDoubleClick{p, ev.Button, ev.Mods}
		ae.depthFirstEv(node, ev2, p)
	case 2:
		ev2 := &event.MouseTripleClick{p, ev.Button, ev.Mods}
		ae.depthFirstEv(node, ev2, p)
	}
}

//----------

type AEDragState struct {
	pressing bool
	node     Node
	point    image.Point
	button   event.MouseButton
	on       bool
	start    bool
}

//----------

type AEMultipleClick struct {
	upTime        time.Time
	downPoint     image.Point
	prevDownPoint image.Point
	action        int // single, double, triple
}
