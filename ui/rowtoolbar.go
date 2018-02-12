package ui

type RowToolbar struct {
	*Toolbar
	Square *RowSquare
}

func NewRowToolbar(row *Row, tb0 *Toolbar) *RowToolbar {
	tb := &RowToolbar{Toolbar: tb0}

	tb.Square = NewRowSquare(row)
	tb.Toolbar.Append(tb.Square)

	return tb
}

func (tb *RowToolbar) CalcChildsBounds() {
	// square size and bounds
	tb.Square.Size = UITheme.RowSquareSize(tb.Theme)
	m := tb.Square.Measure(tb.Bounds.Size())
	sb := tb.Bounds
	sb.Max = sb.Min.Add(m)
	tb.Square.Bounds = sb.Intersect(tb.Bounds)

	// drawer FirstLineOffsetX
	tb.Drawer.Args.FirstLineOffsetX = tb.Square.Size.X
}
