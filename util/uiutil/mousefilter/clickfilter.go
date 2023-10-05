package mousefilter

import (
	"image"
	"time"

	"github.com/jmigpin/editor/util/uiutil/event"
)

// produce click/doubleclick/tripleclick events
type ClickFilter struct {
	m        map[event.MouseButton]*MultipleClick
	emitEvFn func(any, image.Point)
}

func NewClickFilter(emitEvFn func(any, image.Point)) *ClickFilter {
	return &ClickFilter{
		m:        map[event.MouseButton]*MultipleClick{},
		emitEvFn: emitEvFn,
	}
}

func (clickf *ClickFilter) Filter(ev any) {
	switch t := ev.(type) {
	case *event.MouseDown:
		clickf.down(t)
	case *event.MouseUp:
		clickf.up(t)
	case *event.MouseMove:
		clickf.move(t)
	}
}

func (clickf *ClickFilter) down(ev *event.MouseDown) {
	// initialize on demand
	mc, ok := clickf.m[ev.Button]
	if !ok {
		mc = &MultipleClick{}
		clickf.m[ev.Button] = mc
	}

	mc.prevDownPoint = mc.downPoint
	mc.downPoint = ev.Point
}

func (clickf *ClickFilter) up(ev *event.MouseUp) {
	mc, ok := clickf.m[ev.Button]
	if !ok {
		return
	}

	// update time
	upTime0 := mc.upTime
	mc.upTime = time.Now()

	// must be clicked within a margin
	if DetectMove(mc.downPoint, ev.Point) {
		mc.action = MClickActionSingle // reset action
		return
	}

	// if it takes too much time, it gets back to single click
	d := mc.upTime.Sub(upTime0)
	if d > 400*time.Millisecond {
		mc.action = MClickActionSingle
	} else {
		if DetectMove(mc.prevDownPoint, ev.Point) {
			mc.action = MClickActionSingle // reset action
		} else {
			// single, double, triple
			mc.action = (mc.action + 1) % 3
		}
	}

	// always run a click
	ev2 := &event.MouseClick{ev.Point, ev.Button, ev.Buttons, ev.Mods}
	clickf.emitEv(ev2, ev.Point)

	switch mc.action {
	case MClickActionDouble:
		ev2 := &event.MouseDoubleClick{ev.Point, ev.Button, ev.Buttons, ev.Mods}
		clickf.emitEv(ev2, ev.Point)
	case MClickActionTriple:
		ev2 := &event.MouseTripleClick{ev.Point, ev.Button, ev.Buttons, ev.Mods}
		clickf.emitEv(ev2, ev.Point)
	}
}

func (clickf *ClickFilter) move(ev *event.MouseMove) {
	for b, mc := range clickf.m {
		// clear if moved outside move detection margins
		if DetectMove(mc.downPoint, ev.Point) {
			delete(clickf.m, b)
		}
	}
}

//----------

func (clickf *ClickFilter) emitEv(ev any, p image.Point) {
	clickf.emitEvFn(ev, p)
}

//----------

type MultipleClick struct {
	upTime        time.Time
	downPoint     image.Point
	prevDownPoint image.Point
	action        MClickAction
}

type MClickAction int

const (
	MClickActionSingle MClickAction = iota
	MClickActionDouble
	MClickActionTriple
)
