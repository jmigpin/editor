package ui

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil"
)

type UI struct {
	*uiutil.SimpleUI
	Root *Root
}

func NewUI(winName string) (*UI, error) {
	ui := &UI{}

	ui.Root = NewRoot(ui)

	sui, err := uiutil.NewSimpleUI(winName, ui.Root)
	if err != nil {
		return nil, err
	}
	ui.SimpleUI = sui

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

func (ui *UI) ResizeRowToGoodSize(row *Row) {
	if !row.Col.ui.ResizeRowBasedOnPrevRowCursorPosition(row) {
		row.Col.ui.ResizeRowBasedOnPrevRowTextAreaSize(row)
	}
	// ensure up-to-date values now (ex: bounds, drawer.getpoint)
	row.Col.LayoutMarked()
}

func (ui *UI) ResizeRowBasedOnPrevRowTextAreaSize(row *Row) {
	if row.PrevSibling() == nil {
		return
	}
	prevRow := row.PrevSiblingWrapper().(*Row)
	col := row.Col
	colDy := col.Bounds.Dy()

	// percent removed from prevrow
	//prTaDy := prevRow.TextArea.Bounds.Dy()
	//prEndY := prevRow.Bounds.Max.Y - col.Bounds.Min.Y
	//perc := (float64(prEndY) - float64(prTaDy)*2/3) / float64(colDy)
	//col.RowsLayout.Spl.Resize(row, perc)

	// percent of prevrow+row
	taDy := prevRow.TextArea.Bounds.Dy() + row.Bounds.Dy()
	endY := row.Bounds.Max.Y
	perc := (float64(endY-col.Bounds.Min.Y) - float64(taDy)*2/3) / float64(colDy)
	col.RowsLayout.Spl.Resize(row, perc)
}

func (ui *UI) ResizeRowBasedOnPrevRowCursorPosition(row *Row) bool {
	if row.PrevSibling() == nil {
		return false
	}
	prevRow := row.PrevSiblingWrapper().(*Row)
	col := row.Col
	colDy := col.Bounds.Dy()
	lh := row.TextArea.LineHeight()

	ci := prevRow.TextArea.TextCursor.Index()
	p := prevRow.TextArea.GetPoint(ci)
	if p.Y < prevRow.TextArea.Bounds.Min.Y || p.Y+lh > row.Bounds.Max.Y-(row.Toolbar.Bounds.Dy()+lh) {
		return false
	}

	perc := float64(p.Y+lh-col.Bounds.Min.Y) / float64(colDy)
	col.RowsLayout.Spl.Resize(row, perc)

	return true
}

//----------

func (ui *UI) GoodRowPos() *RowPos {
	var best struct {
		r       *image.Rectangle
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
			s := r.TextArea.Bounds.Size()
			a := (s.X * s.Y)

			// after insertion the space will be shared
			a2 := a / 2

			if a2 > best.area {
				best.area = a2
				best.col = c
				best.nextRow = r.NextRow()
			}
		}
	}

	return NewRowPos(best.col, best.nextRow)
}

//----------

type RowPos struct {
	Column  *Column
	NextRow *Row
}

func NewRowPos(col *Column, nextRow *Row) *RowPos {
	return &RowPos{col, nextRow}
}
