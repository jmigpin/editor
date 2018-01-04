package ui

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type ColSeparatorHandle struct {
	*widget.SeparatorHandle
	col *Column
}

func (sh *ColSeparatorHandle) Init(ref widget.Node, col *Column) {
	sh.SeparatorHandle = widget.NewSeparatorHandle(ref)
	sh.SetWrapper(sh)
	sh.col = col
}
func (sh *ColSeparatorHandle) OnInputEvent(ev0 interface{}, p image.Point) bool {
	_ = sh.SeparatorHandle.OnInputEvent(ev0, p)
	if sh.Dragging {
		sh.col.resizeHandleWithSwapToPoint(&p)
	}
	switch ev := ev0.(type) {
	case *event.MouseDown:
		switch ev.Button {
		case event.ButtonWheelLeft:
			sh.col.resizeHandleWithSwapJump(true, &p)
		case event.ButtonWheelRight:
			sh.col.resizeHandleWithSwapJump(false, &p)
		}
	}
	return false
}
