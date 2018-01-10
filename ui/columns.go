package ui

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/widget"
)

type Columns struct {
	widget.EmbedNode
	Layout     *Layout
	ColsLayout *widget.StartPercentLayout // exported to access sp values

	noCols widget.Node
}

func NewColumns(layout *Layout) *Columns {
	cols := &Columns{Layout: layout}

	// when where are no cols, or the first column is pushed aside
	{
		noCols := widget.NewRectangle(layout.UI)
		noCols.Color = &ColumnBgColor
		cols.noCols = noCols
		if ShadowsOn {
			shadow := widget.NewShadow(layout.UI, cols.noCols)
			shadow.Top = ShadowSteps
			shadow.MaxShade = ShadowMaxShade
			cols.noCols = shadow
		}
		cols.Append(cols.noCols)
	}

	cols.ColsLayout = widget.NewStartPercentLayout()
	cols.ColsLayout.MinimumChildSize = 15
	cols.Append(cols.ColsLayout)

	// start with 1 column
	_ = cols.NewColumn()

	return cols
}

func (cols *Columns) NewColumn() *Column {
	col := NewColumn(cols)
	cols.InsertColumnBefore(col, nil)
	return col
}
func (cols *Columns) InsertBefore(col, next widget.Node) {
	panic("!")
}
func (cols *Columns) InsertColumnBefore(col, next *Column) {
	cols.ColsLayout.InsertBefore(col, next)
	cols.CalcChildsBounds()
	cols.MarkNeedsPaint()
}

func (cols *Columns) removeColumn(col *Column) {
	cols.ColsLayout.Remove(col)
	cols.CalcChildsBounds()
	cols.MarkNeedsPaint()

	// ensure one column
	if cols.ColsLayout.ChildsLen() == 0 {
		_ = cols.NewColumn()
	}
}

func (cols *Columns) CalcChildsBounds() {
	cols.EmbedNode.CalcChildsBounds()

	// redimension clear widget to match first row start
	hasCols := cols.ColsLayout.ChildsLen() > 0
	if hasCols {
		x := cols.ColsLayout.FirstChild().Embed().Bounds.Min.X
		cols.noCols.Embed().Bounds.Max.X = x
		cols.noCols.CalcChildsBounds()
	}
}

// Used by restore session.
func (cols *Columns) CloseAllAndOpenN(n int) {
	// close all columns
	cols.ColsLayout.IterChilds(func(c widget.Node) {
		col := c.(*Column)
		col.Close()
	})
	// n new columns (there is already one column ensured)
	for i := 1; i < n; i++ {
		_ = cols.NewColumn()
	}
}

func (cols *Columns) PointColumn(p *image.Point) (*Column, bool) {
	for _, c := range cols.Columns() {
		if p.In(c.Bounds) {
			return c, true
		}
	}
	return nil, false
}
func (cols *Columns) PointColumnExtra(p *image.Point) (*Column, bool) {
	col, ok := cols.PointColumn(p)
	if ok {
		return col, true
	}

	// detect outside of limits, throught X coord
	// assume at least one column is present
	if p.X < cols.FirstChildColumn().Embed().Bounds.Min.X {
		return cols.FirstChildColumn(), true
	} else if p.X > cols.LastChild().Embed().Bounds.Max.X {
		return cols.LastChildColumn(), true
	} else {
		for _, c := range cols.Columns() {
			x0, x1 := c.Bounds.Min.X, c.Bounds.Max.X
			if p.X >= x0 && p.X < x1 {
				return c, true
			}
		}
	}

	return nil, false
}

func (cols *Columns) FirstChildColumn() *Column {
	u := cols.ColsLayout.FirstChild()
	if u == nil {
		return nil
	}
	return u.(*Column)
}
func (cols *Columns) LastChildColumn() *Column {
	u := cols.ColsLayout.LastChild()
	if u == nil {
		return nil
	}
	return u.(*Column)
}

func (cols *Columns) Columns() []*Column {
	u := make([]*Column, 0, cols.ColsLayout.ChildsLen())
	cols.ColsLayout.IterChilds(func(c widget.Node) {
		u = append(u, c.(*Column))
	})
	return u
}
