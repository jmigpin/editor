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
	tb.Square.Size = RowSquareSize(tb.Theme)
	tb.MeasureOpt.FirstLineOffsetX = tb.Square.Size.X

	m := tb.Square.Measure(tb.Bounds.Size())
	r := tb.Bounds
	r.Max = r.Min.Add(m)
	tb.Square.Bounds = r
}
