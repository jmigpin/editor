package widget

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/uiutil/event"
)

// Used by ScrollBar.
type ScrollHandle struct {
	ENode
	ctx    ImageContext
	sb     *ScrollBar
	inside bool
}

func NewScrollHandle(ctx ImageContext, sb *ScrollBar) *ScrollHandle {
	sh := &ScrollHandle{ctx: ctx, sb: sb}

	// the scrollbar handles the decision making, the handle only draws
	sh.AddMarks(MarkNotDraggable)

	return sh
}

func (sh *ScrollHandle) Paint() {
	var c color.Color
	if sh.sb.clicking || sh.sb.dragging {
		c = sh.TreeThemePaletteColor("scrollhandle_select")
	} else if sh.inside {
		c = sh.TreeThemePaletteColor("scrollhandle_hover")
	} else {
		c = sh.TreeThemePaletteColor("scrollhandle_normal")
	}
	imageutil.FillRectangle(sh.ctx.Image(), &sh.Bounds, c)
}

func (sh *ScrollHandle) OnInputEvent(ev interface{}, p image.Point) event.Handle {
	switch ev.(type) {
	case *event.MouseEnter:
		sh.inside = true
		sh.MarkNeedsPaint()
	case *event.MouseLeave:
		sh.inside = false
		sh.MarkNeedsPaint()
	}
	return event.NotHandled
}
