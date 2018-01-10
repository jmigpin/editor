package ui

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/widget"
)

type Column struct {
	*widget.BoxLayout
	Cols       *Columns
	RowsLayout *widget.StartPercentLayout // exported to access sp values

	noRows    widget.Node
	sep       *widget.Rectangle
	sepHandle ColSeparatorHandle
	colSquare *ColumnSquare

	ui *UI
}

func NewColumn(cols *Columns) *Column {
	col := &Column{Cols: cols, ui: cols.Layout.UI}
	col.BoxLayout = widget.NewBoxLayout()

	// separator
	col.sep = widget.NewRectangle(col.ui)
	col.Append(col.sep)
	col.SetChildFill(col.sep, false, true)
	{
		col.sep.Size.X = SeparatorWidth
		col.sep.Color = &SeparatorColor

		col.sepHandle.Init(col.sep, col)
		col.sepHandle.Left = 3
		col.sepHandle.Right = 3
		col.sepHandle.Cursor = widget.WEResizeCursor
		col.Cols.Layout.InsertColSepHandle(&col.sepHandle)
	}

	// content to contain norows and rowslayout
	content := &widget.EmbedNode{}
	col.Append(content)

	// when where are no rows, or the first row is pushed aside
	noRows := widget.NewBoxLayout()
	col.noRows = noRows
	content.Append(col.noRows)
	{
		noRows.YAxis = true

		// square+space box
		ssBox := widget.NewBoxLayout()
		noRows.Append(ssBox)

		// ssBox content
		{
			col.colSquare = NewColumnSquare(col)
			ssBox.Append(col.colSquare)

			space := widget.NewRectangle(col.ui)
			space.Color = &ColumnBgColor
			space.Size = image.Point{5, 15}
			var spaceNode widget.Node = space
			if ShadowsOn {
				shadow := widget.NewShadow(col.ui, spaceNode)
				shadow.Top = ShadowSteps
				shadow.MaxShade = ShadowMaxShade
				spaceNode = shadow
			}
			ssBox.Append(spaceNode)
			ssBox.SetChildFlex(spaceNode, true, false)
			ssBox.SetChildFill(spaceNode, true, true)
		}

		// lower space
		space2 := widget.NewRectangle(col.ui)
		space2.Color = &ColumnBgColor
		noRows.Append(space2)
		noRows.SetChildFlex(space2, true, true)
	}

	// rows layout
	{
		col.RowsLayout = widget.NewStartPercentLayout()
		col.RowsLayout.YAxis = true
		content.Append(col.RowsLayout)
	}

	return col
}

func (col *Column) Close() {
	for _, r := range col.Rows() {
		r.Close()
	}
	col.Cols.Layout.Remove(&col.sepHandle)
	col.Cols.removeColumn(col)
}

func (col *Column) NewRowBefore(next *Row) *Row {
	row := NewRow(col)
	col.insertRowBefore(row, next)
	return row
}

func (col *Column) insertRowBefore(row, next *Row) {
	row.Col = col
	col.RowsLayout.InsertBefore(row, next)
	col.CalcChildsBounds()
	col.MarkNeedsPaint()
}

func (col *Column) removeRow(row *Row) {
	col.RowsLayout.Remove(row)
	col.CalcChildsBounds()
	col.MarkNeedsPaint()
}

func (col *Column) CalcChildsBounds() {
	// update these to handle theme change (based on a row)
	// TODO: needs improvement
	var row *Row
	if col.RowsLayout.ChildsLen() > 0 {
		row = col.RowsLayout.FirstChild().(*Row)
	} else {
		row = NewRow(col)
	}
	col.RowsLayout.MinimumChildSize = row.TextArea.LineHeight()
	col.colSquare.Size = row.Toolbar.Square.Size

	col.BoxLayout.CalcChildsBounds()
	col.sepHandle.CalcChildsBounds()

	// redimension norows widget to match first row start
	hasRows := col.RowsLayout.ChildsLen() > 0
	if hasRows {
		y := col.RowsLayout.FirstChild().Embed().Bounds.Min.Y
		col.noRows.Embed().Bounds.Max.Y = y
		col.noRows.CalcChildsBounds()
	}
}

func (col *Column) FirstChildRow() *Row {
	u := col.RowsLayout.FirstChild()
	if u == nil {
		return nil
	}
	return u.(*Row)
}
func (col *Column) LastChildRow() *Row {
	u := col.RowsLayout.LastChild()
	if u == nil {
		return nil
	}
	return u.(*Row)
}

func (col *Column) Rows() []*Row {
	u := make([]*Row, 0, col.RowsLayout.ChildsLen())
	col.RowsLayout.IterChilds(func(c widget.Node) {
		u = append(u, c.(*Row))
	})
	return u
}

func (col *Column) PointNextRow(p *image.Point) (*Row, bool) {
	for _, r := range col.Rows() {
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

func (col *Column) resizeToPointWithSwap(p *image.Point) {
	bounds := col.Cols.Layout.Bounds
	dx := float64(bounds.Dx())
	perc := float64(p.Sub(bounds.Min).X) / dx

	col.Cols.ColsLayout.ResizeWithMove(col, perc)

	col.Cols.CalcChildsBounds()
	col.Cols.MarkNeedsPaint()
}

func (col *Column) resizeHandleWithMoveJump(left bool, p *image.Point) {
	jump := 20
	if left {
		jump *= -1
	}

	p2 := *p
	p2.X += jump
	col.resizeHandleWithMoveToPoint(&p2)

	p3 := image.Point{col.Bounds.Min.X, p.Y}
	col.ui.WarpPointer(&p3)
}

func (col *Column) resizeHandleWithMoveToPoint(p *image.Point) {
	bounds := col.Cols.Layout.Bounds
	dx := float64(bounds.Dx())
	perc := float64(p.Sub(bounds.Min).X) / dx

	col.Cols.ColsLayout.ResizeWithMove(col, perc)

	col.Cols.CalcChildsBounds()
	col.Cols.MarkNeedsPaint()
}
