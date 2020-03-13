package mousefilter

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/event"
)

// Produce mousedrag* events. Keeps track of the first mouse button used.
type DragFilter struct {
	pressEv  *event.MouseDown
	dragging bool
	emitEvFn func(interface{}, image.Point)
}

func NewDragFilter(emitEvFn func(interface{}, image.Point)) *DragFilter {
	return &DragFilter{emitEvFn: emitEvFn}
}

func (dragf *DragFilter) Filter(ev interface{}) {
	switch t := ev.(type) {
	case *event.MouseDown:
		dragf.keepStartingPoint(t)
	case *event.MouseMove:
		dragf.startOrMove(t)
	case *event.MouseUp:
		dragf.end(t)
	}
}

func (dragf *DragFilter) keepStartingPoint(ev *event.MouseDown) {
	if dragf.pressEv == nil {
		dragf.pressEv = ev
		return
	}
}

func (dragf *DragFilter) startOrMove(ev *event.MouseMove) {
	if dragf.pressEv == nil {
		return
	}
	if !dragf.dragging {
		if DetectMove(dragf.pressEv.Point, ev.Point) {
			dragf.dragging = true
			b := dragf.pressEv.Button
			start := dragf.pressEv.Point
			ev2 := &event.MouseDragStart{start, b, ev.Buttons, ev.Mods}
			dragf.emitEv(ev2, start) // ev.Point is not used
		}
	} else {
		start := dragf.pressEv.Point
		ev2 := &event.MouseDragMove{ev.Point, start, ev.Buttons, ev.Mods}
		dragf.emitEv(ev2, ev.Point)
	}
}

func (dragf *DragFilter) end(ev *event.MouseUp) {
	if dragf.pressEv != nil && ev.Button == dragf.pressEv.Button {
		if dragf.dragging {
			start := dragf.pressEv.Point
			ev2 := &event.MouseDragEnd{ev.Point, start, ev.Button, ev.Buttons, ev.Mods}
			dragf.emitEv(ev2, ev.Point)
		}
		// reset
		dragf.pressEv = nil
		dragf.dragging = false
	}
}

//----------

func (dragf *DragFilter) emitEv(ev interface{}, p image.Point) {
	dragf.emitEvFn(ev, p)
}
