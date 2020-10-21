package ui

import (
	"image"

	"github.com/jmigpin/editor/v2/util/uiutil"
)

type UI struct {
	*uiutil.BasicUI
	Root    *Root
	OnError func(error)
}

func NewUI(winName string) (*UI, error) {
	ui := &UI{}

	ui.Root = NewRoot(ui)

	bui, err := uiutil.NewBasicUI(winName, ui.Root)
	if err != nil {
		return nil, err
	}
	ui.BasicUI = bui

	// set theme before root init
	c1 := &ColorThemeCycler
	c1.Set(c1.CurName, ui.Root)
	c2 := &FontThemeCycler
	c2.Set(c2.CurName, ui.Root)

	// build ui - needs ui.BasicUI to be set
	ui.Root.Init()

	return ui, nil
}

//----------

func (ui *UI) WarpPointerToRectanglePad(r image.Rectangle) {
	p, err := ui.QueryPointer()
	if err != nil {
		return
	}

	pad := 5

	set := func(v *int, min, max int) {
		if max-min < pad*2 {
			*v = min + (max-min)/2
		} else {
			if *v < min+pad {
				*v = min + pad
			} else if *v > max-pad {
				*v = max - pad
			}
		}
	}

	set(&p.X, r.Min.X, r.Max.X)
	set(&p.Y, r.Min.Y, r.Max.Y)

	ui.WarpPointer(p)
}

//----------

func (ui *UI) resizeRowToGoodSize(row *Row) {
	if row.PrevSibling() == nil {
		return
	}
	prevRow := row.PrevSiblingWrapper().(*Row)
	b := ui.rowInsertionBounds(prevRow)
	col := row.Col
	colDy := col.Bounds.Dy()
	perc := float64(b.Min.Y-col.Bounds.Min.Y) / float64(colDy)
	col.RowsLayout.Spl.Resize(row, perc)
}

//----------

func (ui *UI) GoodRowPos() *RowPos {
	var best struct {
		area    int
		col     *Column
		nextRow *Row
	}

	// default position if nothing better is found
	best.col = ui.Root.Cols.FirstChildColumn()

	for _, c := range ui.Root.Cols.Columns() {
		rows := c.Rows()

		// space before first row
		s := c.Bounds.Size()
		if len(rows) > 0 {
			s.Y = rows[0].Bounds.Min.Y - c.Bounds.Min.Y
		}
		a := s.X * s.Y
		if a > best.area {
			best.area = a
			best.col = c
			best.nextRow = nil
			if len(rows) > 0 {
				best.nextRow = rows[0]
			}
		}

		// space between rows
		for _, r := range rows {
			b := ui.rowInsertionBounds(r)
			s := b.Size()
			a := s.X * s.Y
			if a > best.area {
				best.area = a
				best.col = c
				best.nextRow = r.NextRow()
			}
		}
	}

	return NewRowPos(best.col, best.nextRow)
}

//----------

func (ui *UI) rowInsertionBounds(prevRow *Row) image.Rectangle {
	ta := prevRow.TextArea
	if r2, ok := ui.boundsAfterVisibleCursor(ta); ok {
		return *r2
	} else if r3, ok := ui.boundsOfTwoThirds(ta); ok {
		return *r3
	} else {
		b := prevRow.Bounds
		b.Max = b.Max.Sub(b.Size().Div(2)) // half size from StartPercentLayout
		return b
	}
}

func (ui *UI) boundsAfterVisibleCursor(ta *TextArea) (*image.Rectangle, bool) {
	ci := ta.CursorIndex()
	if !ta.IndexVisible(ci) {
		return nil, false
	}
	p := ta.GetPoint(ci)
	lh := ta.LineHeight()
	r := ta.Bounds
	r.Min.Y = p.Y + lh
	r = ta.Bounds.Intersect(r)
	if r.Dy() < lh*2 {
		return nil, false
	}
	return &r, true
}

func (ui *UI) boundsOfTwoThirds(ta *TextArea) (*image.Rectangle, bool) {
	lh := ta.LineHeight()
	if ta.Bounds.Size().Y < lh {
		return nil, false
	}
	b := ta.Bounds
	b.Min.Y = b.Max.Y - (ta.Bounds.Dy() * 2 / 3)
	r := ta.Bounds.Intersect(b)
	return &r, true
}

//----------

func (ui *UI) Error(err error) {
	if ui.OnError != nil {
		ui.OnError(err)
	}
}

//----------

type RowPos struct {
	Column  *Column
	NextRow *Row
}

func NewRowPos(col *Column, nextRow *Row) *RowPos {
	return &RowPos{col, nextRow}
}
