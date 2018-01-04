package ui

import (
	"image"

	"github.com/jmigpin/editor/util/drawutil/hsdrawer"
)

type RowToolbar struct {
	*Toolbar
	OrigColors *hsdrawer.Colors
	Square     *RowSquare
}

func NewRowToolbar(row *Row, tb0 *Toolbar) *RowToolbar {
	tb := &RowToolbar{Toolbar: tb0, OrigColors: tb0.Colors}

	lh := tb0.LineHeight()
	size := image.Point{int(float64(lh) * 0.75), lh}
	tb.Toolbar.MeasureOpt.FirstLineOffsetX = size.X

	tb.Square = NewRowSquare(row)
	tb.Square.Size = size
	tb.Toolbar.Append(tb.Square)

	return tb
}

func (tb *RowToolbar) CalcChildsBounds() {
	m := tb.Square.Measure(tb.Bounds.Size())
	r := tb.Bounds
	r.Max = r.Min.Add(m)
	tb.Square.Bounds = r
}
