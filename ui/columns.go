package ui

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/widget"
)

type Columns struct {
	widget.ENode
	ColsLayout *widget.SplBg // exported to access sp values
	Root       *Root
}

func NewColumns(root *Root) *Columns {
	cols := &Columns{Root: root}

	// when where are no cols, or the first column is pushed aside
	noCols0 := widget.NewRectangle(root.UI)
	noCols0.SetThemePaletteNamePrefix("columns_nocols_")
	noCols := WrapInTopShadowOrSeparator(root.UI, noCols0)

	cols.ColsLayout = widget.NewSplBg(noCols)
	cols.ColsLayout.Spl.MinimumChildSize = 15 // TODO
	cols.Append(cols.ColsLayout)

	return cols
}

func (cols *Columns) NewColumn() *Column {
	col := NewColumn(cols)
	cols.InsertColumnBefore(col, nil)

	// ensure up-to-date values now (ex: bounds, drawer.getpoint)
	cols.LayoutMarked()

	return col
}

func (cols *Columns) InsertBefore(col widget.Node, next *widget.EmbedNode) {
	panic("!")
}

func (cols *Columns) InsertColumnBefore(col, next *Column) {
	var nexte *widget.EmbedNode
	if next != nil {
		nexte = next.Embed()
	}
	cols.ColsLayout.Spl.InsertBefore(col, nexte)
}

func (cols *Columns) removeColumn(col *Column) {
	cols.ColsLayout.Spl.Remove(col)
}

//----------

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
	} else if p.X > cols.LastChildColumn().Embed().Bounds.Max.X {
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

//----------

func (cols *Columns) FirstChildColumn() *Column {
	u := cols.ColsLayout.Spl.FirstChildWrapper()
	if u == nil {
		return nil
	}
	return u.(*Column)
}
func (cols *Columns) LastChildColumn() *Column {
	u := cols.ColsLayout.Spl.LastChildWrapper()
	if u == nil {
		return nil
	}
	return u.(*Column)
}

func (cols *Columns) Columns() []*Column {
	u := make([]*Column, 0, cols.ColsLayout.Spl.ChildsLen())
	cols.ColsLayout.Spl.IterateWrappers2(func(c widget.Node) {
		u = append(u, c.(*Column))
	})
	return u
}
