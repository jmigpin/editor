package ui

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type RowSeparatorHandle struct {
	*widget.SeparatorHandle
	row *Row
}

func NewRowSeparatorHandle(ref widget.Node, row *Row) *RowSeparatorHandle {
	return &RowSeparatorHandle{
		SeparatorHandle: widget.NewSeparatorHandle(ref),
		row:             row,
	}
}
func (sh *RowSeparatorHandle) OnInputEvent(ev0 interface{}, p image.Point) bool {
	h := sh.SeparatorHandle.OnInputEvent(ev0, p)
	if sh.Dragging {
		sh.row.resizeWithMoveToPoint(&p)
	}
	switch ev := ev0.(type) {
	case *event.MouseDown:
		switch ev.Button {
		case event.ButtonWheelUp:
			sh.row.resizeWithPushJump(true, &p)
		case event.ButtonWheelDown:
			sh.row.resizeWithPushJump(false, &p)
		}
	case *event.MouseClick:
		switch ev.Button {
		case event.ButtonMiddle:
			sh.row.Close()
		}
	}
	return true || h // handled, no other widget will get the event
}
