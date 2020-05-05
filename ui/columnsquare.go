package ui

import (
	"image"

	"github.com/jmigpin/editor/v2/util/imageutil"
	"github.com/jmigpin/editor/v2/util/uiutil/event"
	"github.com/jmigpin/editor/v2/util/uiutil/widget"
)

type ColumnSquare struct {
	widget.ENode
	Size image.Point
	col  *Column
}

func NewColumnSquare(col *Column) *ColumnSquare {
	sq := &ColumnSquare{col: col, Size: image.Point{5, 5}}
	sq.Cursor = event.CloseCursor
	return sq
}

func (sq *ColumnSquare) Measure(hint image.Point) image.Point {
	return imageutil.MinPoint(sq.Size, hint)
}
func (sq *ColumnSquare) Paint() {
	c := sq.TreeThemePaletteColor("columnsquare")
	imageutil.FillRectangle(sq.col.ui.Image(), sq.Bounds, c)
}
func (sq *ColumnSquare) OnInputEvent(ev interface{}, p image.Point) event.Handled {
	switch t := ev.(type) {
	case *event.MouseClick:
		switch t.Button {
		case event.ButtonLeft, event.ButtonMiddle, event.ButtonRight:
			sq.col.Close()
		}
	}
	return true
}
