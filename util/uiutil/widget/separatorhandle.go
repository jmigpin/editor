package widget

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/event"
)

// A transparent widget added to a top layer (usually multilayer) to facilitate dragging.
// Calculations are made on top of the reference node (usually a thin separator that otherwise would not be easy to put the pointer over for dragging).
type SeparatorHandle struct {
	ENode
	Top, Bottom, Left, Right int
	Dragging                 bool

	ref Node // reference node for calc bounds
}

func NewSeparatorHandle(ref Node) *SeparatorHandle {
	sh := &SeparatorHandle{ref: ref}
	sh.AddMarks(MarkNotPaintable)
	return sh
}

func (sh *SeparatorHandle) Measure(hint image.Point) image.Point {
	panic("calling measure on thin separator handle")
}

func (sh *SeparatorHandle) Layout() {
	// calc own bounds based on reference node
	b := sh.ref.Embed().Bounds
	b.Min.X -= sh.Left
	b.Max.X += sh.Right
	b.Min.Y -= sh.Top
	b.Max.Y += sh.Bottom

	// limit with parents bounds (might be wider/thiner)
	pb := sh.Parent.Bounds
	b = b.Intersect(pb)

	// set own bounds
	sh.Bounds = b
}

func (sh *SeparatorHandle) OnInputEvent(ev0 interface{}, p image.Point) event.Handle {
	switch ev := ev0.(type) {
	case *event.MouseDown:
		switch ev.Button {
		case event.ButtonLeft:
			sh.Dragging = true
		}
	case *event.MouseUp:
		switch ev.Button {
		case event.ButtonLeft:
			if sh.Dragging {
				sh.Dragging = false
			}
		}

	// mouseup might not be triggered if moving too fast, but dragend will
	case *event.MouseDragEnd:
		if sh.Dragging {
			sh.Dragging = false
		}
	}

	return sh.ref.Embed().Wrapper.OnInputEvent(ev0, p)
}
