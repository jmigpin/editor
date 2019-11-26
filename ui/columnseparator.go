package ui

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type ColSeparator struct {
	*widget.Separator
	col *Column
}

func NewColSeparator(col *Column) *ColSeparator {
	sep := widget.NewSeparator(col.ui, col.Cols.Root.MultiLayer)
	sep.Size.X = separatorWidth
	sep.Handle.Left = 3
	sep.Handle.Right = 3
	sep.Handle.Cursor = event.WEResizeCursor

	csep := &ColSeparator{Separator: sep, col: col}
	csep.SetThemePaletteNamePrefix("colseparator_")
	return csep
}
func (sh *ColSeparator) OnInputEvent(ev0 interface{}, p image.Point) event.Handled {
	if sh.Handle.Dragging {
		sh.col.resizeWithMoveToPoint(&p)
	}
	switch ev := ev0.(type) {
	case *event.MouseDown:
		switch ev.Button {
		case event.ButtonWheelLeft:
			sh.col.resizeWithMoveJump(true, &p)
		case event.ButtonWheelRight:
			sh.col.resizeWithMoveJump(false, &p)
		}
	}
	return event.HTrue // no other widget will get the event
}
