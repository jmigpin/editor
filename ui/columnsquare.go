package ui

import (
	"image"

	"github.com/jmigpin/editor/imageutil"
	"github.com/jmigpin/editor/uiutil/event"
	"github.com/jmigpin/editor/uiutil/widget"
)

type ColumnSquare struct {
	widget.EmbedNode
	Size image.Point
	col  *Column
}

func NewColumnSquare(col *Column) *ColumnSquare {
	sq := &ColumnSquare{col: col, Size: image.Point{5, 5}}
	sq.Cursor = widget.CloseCursor
	return sq
}

func (sq *ColumnSquare) Measure(hint image.Point) image.Point {
	return widget.MinPoint(sq.Size, hint)
}
func (sq *ColumnSquare) Paint() {
	b := sq.Bounds
	img := sq.col.ui.Image()
	imageutil.FillRectangle(img, &b, SquareColor)
}
func (sq *ColumnSquare) OnInputEvent(ev interface{}, p image.Point) bool {
	switch ev.(type) {
	case *event.MouseClick:
		sq.col.Close()
	}
	return true
}
