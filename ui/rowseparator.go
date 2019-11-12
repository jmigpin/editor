package ui

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type RowSeparator struct {
	*widget.Separator
	row *Row
}

func NewRowSeparator(row *Row) *RowSeparator {
	sep := widget.NewSeparator(row.ui, row.Col.Cols.Root.MultiLayer)
	sep.Size.Y = separatorWidth
	sep.Handle.Top = 3
	sep.Handle.Bottom = 3
	sep.Handle.Cursor = event.MoveCursor

	rsep := &RowSeparator{Separator: sep, row: row}
	rsep.SetThemePaletteNamePrefix("rowseparator_")
	return rsep
}
func (sh *RowSeparator) OnInputEvent(ev0 interface{}, p image.Point) event.Handle {
	if sh.Handle.Dragging {
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
	return event.Handled //no other widget will get the event
}
