package ui

import (
	"image"

	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type RowSquare struct {
	widget.EmbedNode
	Size  image.Point
	row   *Row
	state RowState
}

func NewRowSquare(row *Row) *RowSquare {
	sq := &RowSquare{row: row, Size: image.Point{5, 5}}
	sq.Cursor = widget.CloseCursor
	return sq
}
func (sq *RowSquare) Measure(hint image.Point) image.Point {
	return imageutil.MinPoint(sq.Size, hint)
}

func (sq *RowSquare) Paint() {
	img := sq.row.ui.Image()

	// background
	bg := sq.TreeThemePaletteColor("noselection_bg")
	if sq.state.has(EditedRowState) {
		bg = sq.TreeThemePaletteColor("rs_edited")
	}
	if sq.state.has(NotExistRowState) {
		bg = sq.TreeThemePaletteColor("rs_not_exist")
	}
	if sq.state.has(ExecutingRowState) {
		bg = sq.TreeThemePaletteColor("rs_executing")
	}
	imageutil.FillRectangle(img, &sq.Bounds, bg)

	// mini-squares
	if sq.state.has(ActiveRowState) {
		r := sq.miniSq(0)
		c := sq.TreeThemePaletteColor("rs_active")
		imageutil.FillRectangle(img, &r, c)
	}
	if sq.state.has(DiskChangesRowState) {
		r := sq.miniSq(1)
		c := sq.TreeThemePaletteColor("rs_disk_changes")
		imageutil.FillRectangle(img, &r, c)
	}
	if sq.state.has(DuplicateRowState) {
		r := sq.miniSq(2)
		c := sq.TreeThemePaletteColor("rs_duplicate")
		imageutil.FillRectangle(img, &r, c)
	}
	if sq.state.has(DuplicateHighlightRowState) {
		r := sq.miniSq(2)
		c := sq.TreeThemePaletteColor("rs_duplicate_highlight")
		imageutil.FillRectangle(img, &r, c)
	}
	if sq.state.has(AnnotationsRowState) {
		r := sq.miniSq(3)
		c := sq.TreeThemePaletteColor("rs_annotations")
		imageutil.FillRectangle(img, &r, c)
	}
	if sq.state.has(AnnotationsEditedRowState) {
		r := sq.miniSq(3)
		c := sq.TreeThemePaletteColor("rs_annotations_edited")
		imageutil.FillRectangle(img, &r, c)
	}
}
func (sq *RowSquare) miniSq(i int) image.Rectangle {
	// mini squares
	// [0,1]
	// [2,3]

	// mini square rectangle
	maxXI, maxYI := 1, 1
	sideX, sideY := sq.Size.X/(maxXI+1), sq.Size.Y/(maxYI+1)
	x, y := i%2, i/2
	r := image.Rect(0, 0, sideX, sideY)
	r = r.Add(image.Point{x * sideX, y * sideY})

	// avoid rounding errors
	if x == maxXI {
		r.Max.X = sq.Size.X
	}
	if y == maxYI {
		r.Max.Y = sq.Size.Y
	}

	// mini square position
	r2 := r.Add(sq.Bounds.Min).Intersect(sq.Bounds)

	return r2
}

func (sq *RowSquare) SetState(s RowState, v bool) {
	u := sq.state.has(s)
	if u != v {
		sq.state.set(s, v)
		sq.MarkNeedsPaint()
	}
}
func (sq *RowSquare) HasState(s RowState) bool {
	return sq.state.has(s)
}
func (sq *RowSquare) OnInputEvent(ev interface{}, p image.Point) bool {
	switch ev.(type) {
	case *event.MouseClick:
		sq.row.Close()
	}
	return true
}

type RowState uint16

func (m *RowState) add(u RowState)      { *m |= u }
func (m *RowState) remove(u RowState)   { *m &^= u }
func (m *RowState) has(u RowState) bool { return (*m)&u > 0 }
func (m *RowState) set(u RowState, v bool) {
	if v {
		m.add(u)
	} else {
		m.remove(u)
	}
}

const (
	ActiveRowState RowState = 1 << iota
	ExecutingRowState
	EditedRowState
	DiskChangesRowState
	NotExistRowState
	DuplicateRowState
	DuplicateHighlightRowState
	AnnotationsRowState
	AnnotationsEditedRowState
)
