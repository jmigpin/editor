package ui

import (
	"image"

	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
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
	return imageutil.MinPoint(sq.Size, hint)
}
func (sq *ColumnSquare) Paint() {
	bg := sq.TreeThemePaletteColor("noselection_bg")
	imageutil.FillRectangle(sq.col.ui.Image(), &sq.Bounds, bg)
}
func (sq *ColumnSquare) OnInputEvent(ev interface{}, p image.Point) bool {
	switch ev.(type) {
	case *event.MouseClick:
		sq.col.Close()
	}
	return true
}
