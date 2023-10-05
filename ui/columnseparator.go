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
func (sh *ColSeparator) OnInputEvent(ev0 any, p image.Point) event.Handled {
	switch ev := ev0.(type) {
	case *event.MouseDragMove:
		switch {
		case ev.Buttons.Is(event.ButtonLeft):
			p.X += sh.Handle.DragPad.X
			sh.col.resizeWithMoveToPoint(&p)
		}
	case *event.MouseDown:
		switch ev.Button {
		case event.ButtonWheelLeft:
			sh.col.resizeWithMoveJump(true, &p)
		case event.ButtonWheelRight:
			sh.col.resizeWithMoveJump(false, &p)
		}
	}
	return true // no other widget will get the event
}
