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
	t := UITheme
	_, bg := t.NoSelectionColors(&t.ToolbarTheme)
	if sq.state.has(EditedRowState) {
		bg = t.RowSquare.Edited
	}
	if sq.state.has(NotExistRowState) {
		bg = t.RowSquare.NotExist
	}
	if sq.state.has(ExecutingRowState) {
		bg = t.RowSquare.Executing
	}
	imageutil.FillRectangle(img, &sq.Bounds, bg)

	// mini-squares
	if sq.state.has(ActiveRowState) {
		r := sq.miniSq(0)
		imageutil.FillRectangle(img, &r, t.RowSquare.Active)
	}
	if sq.state.has(DiskChangesRowState) {
		r := sq.miniSq(1)
		imageutil.FillRectangle(img, &r, t.RowSquare.DiskChanges)
	}
	if sq.state.has(DuplicateRowState) {
		r := sq.miniSq(2)
		imageutil.FillRectangle(img, &r, t.RowSquare.Duplicate)
	}
	if sq.state.has(DuplicateHighlightRowState) {
		r := sq.miniSq(2)
		imageutil.FillRectangle(img, &r, t.RowSquare.DuplicateHighlight)
	}
	if sq.state.has(AnnotationsRowState) {
		r := sq.miniSq(3)
		imageutil.FillRectangle(img, &r, t.RowSquare.Annotations)
	}
	if sq.state.has(AnnotationsEditedRowState) {
		r := sq.miniSq(3)
		imageutil.FillRectangle(img, &r, t.RowSquare.AnnotationsEdited)
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
