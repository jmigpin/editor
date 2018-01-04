package ui

import (
	"image"
	"image/color"

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
	return widget.MinPoint(sq.Size, hint)
}

func (sq *RowSquare) Paint() {
	b := sq.Bounds
	img := sq.row.ui.Image()

	// background
	var bg color.Color = SquareColor
	if sq.state.has(EditedRowState) {
		bg = SquareEditedColor
	}
	if sq.state.has(NotExistRowState) {
		bg = SquareNotExistColor
	}
	if sq.state.has(ExecutingRowState) {
		bg = SquareExecutingColor
	}
	imageutil.FillRectangle(img, &b, bg)

	// mini-squares
	if sq.state.has(ActiveRowState) {
		r := sq.miniSq(0)
		imageutil.FillRectangle(img, &r, SquareActiveColor)
	}
	if sq.state.has(DiskChangesRowState) {
		r := sq.miniSq(1)
		imageutil.FillRectangle(img, &r, SquareDiskChangesColor)
	}
	if sq.state.has(DuplicateRowState) {
		r := sq.miniSq(2)
		imageutil.FillRectangle(img, &r, SquareDuplicateColor)
	}
	if sq.state.has(HighlightDuplicateRowState) {
		r := sq.miniSq(2)
		imageutil.FillRectangle(img, &r, SquareHighlightDuplicateColor)
	}
}
func (sq *RowSquare) miniSq(i int) image.Rectangle {
	// [0,1]
	// [2,3]
	sideX := sq.Size.X / 2
	sideY := sq.Size.Y / 2
	r := image.Rect(0, 0, sideX, sideY)
	x := i % 2
	y := i / 2
	r = r.Add(image.Point{x * sideX, y * sideY}).Add(sq.Bounds.Min)
	r = r.Intersect(sq.Bounds)

	// avoid rounding errors
	if x == 1 {
		r.Max.X = sq.Bounds.Max.X
	}
	if y == 1 {
		r.Max.Y = sq.Bounds.Max.Y
	}

	return r
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

type RowState uint8

func (m *RowState) add(u RowState)      { *m |= u }
func (m *RowState) remove(u RowState)   { *m &^= u }
func (m *RowState) has(u RowState) bool { return *m&u > 0 }
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
	HighlightDuplicateRowState
)
