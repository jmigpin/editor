package ui

import (
	"image"

	"github.com/jmigpin/editor/util/evreg"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type Column struct {
	*widget.BoxLayout
	RowsLayout *widget.SplBg // exported to access sp values
	Cols       *Columns
	EvReg      evreg.Register

	sep       *ColSeparator
	colSquare *ColumnSquare
	ui        *UI
}

func NewColumn(cols *Columns) *Column {
	col := &Column{Cols: cols, ui: cols.Root.UI}
	col.BoxLayout = widget.NewBoxLayout()

	// separator
	col.sep = NewColSeparator(col)
	col.Append(col.sep)
	col.SetChildFill(col.sep, false, true)

	// when where are no rows, or the first row is pushed aside
	noRows0 := widget.NewFreeLayout()
	noRows := WrapInTopShadowOrSeparator(cols.Root.UI, noRows0)
	{
		noRows0.SetThemePaletteNamePrefix("column_norows_")
		rect := widget.NewRectangle(col.ui)
		rect.Size = image.Point{10000, 10000}
		col.colSquare = NewColumnSquare(col)
		noRows0.Append(rect, col.colSquare)
	}

	// rows layout
	col.RowsLayout = widget.NewSplBg(noRows)
	col.RowsLayout.Spl.YAxis = true
	col.Append(col.RowsLayout)

	return col
}

func (col *Column) Close() {
	for _, r := range col.Rows() {
		r.Close()
	}
	col.Cols.removeColumn(col)
	col.Cols = nil
	col.sep.Close()
	col.EvReg.RunCallbacks(ColumnCloseEventId, &ColumnCloseEvent{col})
}

func (col *Column) NewRowBefore(next *Row) *Row {
	row := NewRow(col)
	col.insertRowBefore(row, next)
	return row
}

func (col *Column) insertRowBefore(row, next *Row) {
	var nexte *widget.EmbedNode
	if next != nil {
		nexte = next.Embed()
	}

	row.Col = col
	col.RowsLayout.Spl.InsertBefore(row, nexte)

	// resizing before laying out (previous row still has the old bounds)
	col.ui.resizeRowToGoodSize(row)

	// ensure up-to-date values now (ex: bounds, drawer.getpoint)
	col.LayoutMarked()
}

func (col *Column) removeRow(row *Row) {
	col.RowsLayout.Spl.Remove(row)
}

func (col *Column) Layout() {
	tf := col.TreeThemeFont()
	col.RowsLayout.Spl.MinimumChildSize = UIThemeUtil.RowMinimumHeight(tf)
	col.colSquare.Size = UIThemeUtil.RowSquareSize(tf)

	col.BoxLayout.Layout()
}

//----------

func (col *Column) FirstChildRow() *Row {
	u := col.RowsLayout.Spl.FirstChildWrapper()
	if u == nil {
		return nil
	}
	return u.(*Row)
}
func (col *Column) LastChildRow() *Row {
	u := col.RowsLayout.Spl.LastChildWrapper()
	if u == nil {
		return nil
	}
	return u.(*Row)
}

//----------

func (col *Column) Rows() []*Row {
	u := make([]*Row, 0, col.RowsLayout.Spl.ChildsLen())
	col.RowsLayout.Spl.IterateWrappers2(func(c widget.Node) {
		u = append(u, c.(*Row))
	})
	return u
}

//----------

func (col *Column) PointNextRow(p *image.Point) (*Row, bool) {
	for _, r := range col.Rows() {
		if p.Y < r.Bounds.Min.Y {
			return r, true
		}
		if p.In(r.Bounds) {
			return r.NextRow(), true
		}
	}
	return nil, false
}

func (col *Column) PointNextRowExtra(p *image.Point) (*Row, bool) {
	next, ok := col.PointNextRow(p)
	if ok {
		return next, true
	}

	first := col.FirstChildRow()
	if first == nil {
		return nil, true
	}
	last := col.LastChildRow()
	if p.Y < first.Embed().Bounds.Min.Y {
		return first, true
	} else if p.Y > last.Embed().Bounds.Max.Y {
		return nil, true
	} else {
		for _, r := range col.Rows() {
			y0, y1 := r.Bounds.Min.Y, r.Bounds.Max.Y
			if y0 <= p.Y && p.Y < y1 {
				return r.NextRow(), true
			}
		}
	}

	return nil, false
}

//----------

func (col *Column) resizeToPointWithSwap(p *image.Point) {
	bounds := col.Cols.Root.Bounds
	dx := float64(bounds.Dx())
	perc := float64(p.Sub(bounds.Min).X) / dx

	col.Cols.ColsLayout.Spl.ResizeWithMove(col, perc)
}

func (col *Column) resizeWithMoveJump(left bool, p *image.Point) {
	jump := 20
	if left {
		jump *= -1
	}

	p2 := *p
	p2.X += jump
	col.resizeWithMoveToPoint(&p2)

	// layout for accurate col.bounds to warp pointer
	col.Cols.ColsLayout.Spl.Layout()

	p3 := image.Point{col.Bounds.Min.X, p.Y}
	col.ui.WarpPointer(p3)
}

func (col *Column) resizeWithMoveToPoint(p *image.Point) {
	bounds := col.Cols.Root.Bounds
	dx := float64(bounds.Dx())
	perc := float64(p.Sub(bounds.Min).X) / dx

	col.Cols.ColsLayout.Spl.ResizeWithMove(col, perc)
}

//----------

const (
	ColumnCloseEventId = iota
)

type ColumnCloseEvent struct {
	Col *Column
}
