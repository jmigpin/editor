package ui

import (
	"image"

	"github.com/jmigpin/editor/uiutil/widget"
)

type Layout struct {
	widget.FlowLayout
	UI      *UI
	Toolbar *Toolbar
	Cols    *Columns
}

func NewLayout(ui *UI) *Layout {
	layout := &Layout{}
	layout.UI = ui

	layout.Toolbar = NewToolbar(ui, layout)
	layout.Toolbar.SetExpand(true, false)

	sep := widget.NewSpace(ui)
	sep.SetExpand(true, false)
	sep.Size.Y = SeparatorWidth
	sep.Color = SeparatorColor

	layout.Cols = NewColumns(layout)
	layout.Cols.SetExpand(true, true)

	layout.YAxis = true
	widget.AppendChilds(layout, layout.Toolbar, sep, layout.Cols)

	return layout
}

func (l *Layout) GoodColumnRowPlace() (*Column, *Row) {

	// TODO: accept optional row, or take into consideration active row
	// TODO: don't go too far away, stay close (active row)

	var best struct {
		r       *image.Rectangle
		area    int
		col     *Column
		nextRow *Row
	}

	for _, c := range l.Cols.Columns() {
		rows := c.Rows()
		if len(rows) == 0 {
			s := c.Bounds().Size()
			a := s.X * s.Y
			if a > best.area {
				best.area = a
				best.col = c
				best.nextRow = nil
			}
		} else {
			for _, r := range rows {
				s := r.Bounds().Size()
				a := (s.X * s.Y)

				// current end percent inserts rows and shares space
				// with prev row, hence div by 2
				a2 := a / 2

				if a2 > best.area {
					best.area = a2
					best.col = c
					best.nextRow = nil
					r2, ok := r.NextRow()
					if ok {
						best.nextRow = r2
					}
				}
			}
		}
	}

	return best.col, best.nextRow
}

// TODO: remove - here for reference only
//func (cols *Columns) ColumnWithGoodPlaceForNewRow() *Column {
//	var best struct {
//		r    *image.Rectangle
//		area int
//		col  *Column
//	}

//	u, ok := cols.FirstChildColumn()
//	if ok {
//		best.col = u
//	}

//	rectArea := func(r *image.Rectangle) int {
//		p := r.Size()
//		return p.X * p.Y
//	}
//	columns := cols.Columns()
//	for _, col := range columns {
//		dy0 := col.Bounds().Dy()
//		dy := dy0 / (len(columns) + 1)

//		// take into consideration the textarea content size
//		usedY := 0
//		for _, r := range col.Rows() {
//			ry := r.Bounds().Dy()

//			// small text - count only needed height
//			ry1 := ry - r.TextArea.Bounds().Dy()
//			ry2 := ry1 + r.TextArea.StrHeight().Round()
//			if ry2 < ry {
//				ry = ry2
//			}

//			usedY += ry
//		}
//		dy2 := dy0 - usedY
//		if dy < dy2 {
//			dy = dy2
//		}

//		r := image.Rect(0, 0, col.Bounds().Dx(), dy)
//		area := rectArea(&r)
//		if area > best.area {
//			best.area = area
//			best.r = &r
//			best.col = col
//		}
//	}
//	if best.col == nil {
//		panic("col is nil")
//	}
//	return best.col
//}
