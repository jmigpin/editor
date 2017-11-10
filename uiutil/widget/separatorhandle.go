package widget

import (
	"image"

	"github.com/jmigpin/editor/uiutil/event"
)

// A transparent widget added to a top layer (usually multilayer) to facilitate dragging.
// Calculations are made on top of the reference node (usually a thin separator that otherwise would not be easy to put the pointer over for dragging).
type SeparatorHandle struct {
	LeafEmbedNode
	Top, Bottom, Left, Right int
	Cursor                   Cursor
	Dragging                 bool

	ref       Node // reference node for calc bounds
	ctx       Context
	cursorRef *CursorRef
}

func (sh *SeparatorHandle) Init(ctx Context, ref Node) {
	*sh = SeparatorHandle{ctx: ctx, ref: ref}
	sh.SetWrapper(sh)
	sh.SetNotPaintable(true)
}
func (sh *SeparatorHandle) Measure(hint image.Point) image.Point {
	panic("calling measure on thin separator handle")
}
func (sh *SeparatorHandle) CalcChildsBounds() {
	// set own bounds
	b := sh.ref.Bounds()
	b.Min.X -= sh.Left
	b.Max.X += sh.Right
	b.Min.Y -= sh.Top
	b.Max.Y += sh.Bottom

	// limit with parents bounds
	pb := sh.Parent().Bounds()
	b = b.Intersect(pb)

	sh.LeafEmbedNode.SetBounds(&b)
}
func (sh *SeparatorHandle) Paint() {
}
func (sh *SeparatorHandle) OnInputEvent(ev0 interface{}, p image.Point) bool {
	switch ev := ev0.(type) {
	case *event.MouseEnter:
		sh.SetPointerCursor(sh.ctx, sh.Cursor)
	case *event.MouseLeave:
		sh.UnsetPointerCursor(sh.ctx)

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

	case *event.MouseDragEnd:
		if sh.Dragging {
			sh.Dragging = false
		}
	}
	return false
}
