package ui

import (
	"github.com/jmigpin/editor/util/drawutil/drawer3"
)

type RowToolbar struct {
	*Toolbar
	Square *RowSquare
}

func NewRowToolbar(row *Row) *RowToolbar {
	tb0 := NewToolbar(row.ui)

	tb := &RowToolbar{Toolbar: tb0}

	tb.Square = NewRowSquare(row)
	tb.Append(tb.Square)

	return tb
}

func (tb *RowToolbar) Layout() {
	// TODO: should use freelayout instead to set the square position

	// square
	m := tb.Square.Measure(tb.Bounds.Size())
	sqb := tb.Bounds
	sqb.Max = sqb.Min.Add(m)
	tb.Square.Bounds = sqb.Intersect(tb.Bounds)

	tb.Toolbar.Layout()
}

func (tb *RowToolbar) OnThemeChange() {
	tb.Toolbar.OnThemeChange()
	tb.Square.Size = UIThemeUtil.RowSquareSize(tb.TreeThemeFont())

	if d, ok := tb.Drawer.(*drawer3.PosDrawer); ok {
		d.SetFirstLineOffsetX(tb.Square.Size.X)
	}
}
