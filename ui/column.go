package ui

import (
	"image"

	"github.com/jmigpin/editor/imageutil"
	"github.com/jmigpin/editor/uiutil/event"
	"github.com/jmigpin/editor/uiutil/widget"
)

type Column struct {
	*widget.FlowLayout
	Square     *Square
	Cols       *Columns
	RowsLayout *widget.EndPercentLayout

	sep       widget.Rectangle
	sepHandle ColSeparatorHandle
	sqc       *widget.FlowLayout // square container (show/hide)

	ui *UI
}

func NewColumn(cols *Columns) *Column {
	col := &Column{Cols: cols, ui: cols.Layout.UI}
	col.FlowLayout = widget.NewFlowLayout()
	col.SetWrapper(col)

	col.Square = NewSquare(col.ui)
	col.Square.EvReg.Add(SquareInputEventId, col.onSquareInput)

	col.sep.Init(col.ui)
	col.sep.SetExpand(false, true)
	col.sep.Size.X = SeparatorWidth
	col.sep.Color = &SeparatorColor

	col.sepHandle.Init(col.ui, &col.sep, col)
	col.sepHandle.Left = 3
	col.sepHandle.Right = 3
	col.sepHandle.Cursor = widget.WEResizeCursor
	col.Cols.Layout.InsertColSepHandle(&col.sepHandle)

	col.RowsLayout = widget.NewEndPercentLayout()
	col.RowsLayout.YAxis = true

	// square (when there are no rows)
	col.sqc = widget.NewFlowLayout()
	var sqBorder widget.Pad
	sqBorder.Init(col.ui, col.Square)
	sqBorder.Color = &RowInnerSeparatorColor
	sqBorder.Bottom = SeparatorWidth

	var space widget.Rectangle
	space.Init(col.ui)
	space.SetFill(true, true)

	var spaceNode widget.Node = &space
	if ShadowsOn {
		// innershadow bellow the toolbar
		var shadow widget.Shadow
		shadow.Init(col.ui, &space)
		shadow.Top = ShadowSteps
		shadow.MaxShade = ShadowMaxShade
		spaceNode = &shadow

		sqBorder.Top = SeparatorWidth
	}

	sqBorder.Right = SeparatorWidth
	col.sqc.Append(&sqBorder, spaceNode)

	rightSide := widget.NewFlowLayout()
	rightSide.YAxis = true
	rightSide.Append(col.sqc, col.RowsLayout)

	col.Append(&col.sep, rightSide)

	return col
}
func (col *Column) Close() {
	for _, r := range col.Rows() {
		r.Close()
	}
	col.Cols.Layout.Remove(&col.sepHandle)
	col.Cols.removeColumn(col)
}
func (col *Column) Paint() {
	if len(col.RowsLayout.Childs()) == 0 {
		b := col.Bounds()
		imageutil.FillRectangle(col.ui.Image(), &b, ColumnBgColor)
		return
	}
}

func (col *Column) NewRowBefore(next *Row) *Row {
	row := NewRow(col)
	col.insertBefore(row, next)
	return row
}

func (col *Column) insertBefore(row, next *Row) {
	row.Col = col
	if next == nil {
		col.RowsLayout.PushBack(row)
	} else {
		col.RowsLayout.InsertBefore(row, next)
	}
	col.CalcChildsBounds()
	col.MarkNeedsPaint()
}

func (col *Column) removeRow(row *Row) {
	col.RowsLayout.Remove(row)
	col.CalcChildsBounds()
	col.MarkNeedsPaint()
}

func (col *Column) CalcChildsBounds() {
	col.fixSquareVisibility()
	col.FlowLayout.CalcChildsBounds()
	col.sepHandle.CalcChildsBounds()
}

func (col *Column) fixSquareVisibility() {
	// hide/show column square if we have a first row
	_, ok := col.FirstChildRow()
	hide := ok
	if col.sqc.Hidden() != hide {
		col.sqc.SetHidden(hide)
		col.MarkNeedsPaint()
	}
}

func (col *Column) onSquareInput(ev0 interface{}) {
	sqEv := ev0.(*SquareInputEvent)
	switch ev := sqEv.Event.(type) {
	case *event.MouseEnter:
		col.SetPointerCursor(col.ui, widget.CloseCursor)
	case *event.MouseLeave:
		col.UnsetPointerCursor(col.ui)
	case *event.MouseClick:
		switch ev.Button {
		case event.ButtonLeft:
			col.UnsetPointerCursor(col.ui)
			col.Cols.CloseColumnEnsureOne(col)
		}
	}
}

func (col *Column) FirstChildRow() (*Row, bool) {
	u := col.RowsLayout.FirstChild()
	if u == nil {
		return nil, false
	}
	return u.(*Row), true
}
func (col *Column) NextColumn() (*Column, bool) {
	u := col.Next()
	if u == nil {
		return nil, false
	}
	return u.(*Column), true
}
func (col *Column) PrevColumn() (*Column, bool) {
	u := col.Prev()
	if u == nil {
		return nil, false
	}
	return u.(*Column), true
}
func (col *Column) Rows() []*Row {
	childs := col.RowsLayout.Childs()
	u := make([]*Row, 0, len(childs))
	for _, h := range childs {
		u = append(u, h.(*Row))
	}
	return u
}

func (col *Column) PointRow(p *image.Point) (*Row, bool) {
	for _, r := range col.Rows() {
		if p.In(r.Bounds()) {
			return r, true
		}
	}
	return nil, false
}

func (col *Column) resizeToPointWithSwap(p *image.Point) {
	bounds := col.Cols.Layout.Bounds()
	dx := float64(bounds.Dx())
	perc := float64(p.Sub(bounds.Min).X) / dx
	min := 30 / dx

	percIsLeft := ScrollbarLeft
	col.Cols.ResizeEndPercentWithSwap(col, perc, percIsLeft, min)

	col.Cols.CalcChildsBounds()
	col.Cols.MarkNeedsPaint()
}

func (col *Column) resizeHandleWithSwapJump(left bool, p *image.Point) {
	jump := 20
	if left {
		jump *= -1
	}

	p2 := *p
	p2.X += jump
	col.resizeHandleWithSwapToPoint(&p2)

	p3 := image.Point{col.Bounds().Min.X, p.Y}
	col.ui.WarpPointer(&p3)
}
func (col *Column) resizeHandleWithSwapToPoint(p *image.Point) {
	bounds := col.Cols.Layout.Bounds()
	dx := float64(bounds.Dx())
	perc := float64(p.Sub(bounds.Min).X) / dx
	min := 30 / dx

	// column handle is positioned on the left (beginning) of the column
	percIsLeft := true // always on the left

	col.Cols.ResizeEndPercentWithSwap(col, perc, percIsLeft, min)

	col.Cols.CalcChildsBounds()
	col.Cols.MarkNeedsPaint()
}

type ColSeparatorHandle struct {
	widget.SeparatorHandle
	col *Column
}

func (sh *ColSeparatorHandle) Init(ctx widget.Context, ref widget.Node, col *Column) {
	sh.SeparatorHandle.Init(ctx, ref)
	sh.SetWrapper(sh)
	sh.col = col
}
func (sh *ColSeparatorHandle) OnInputEvent(ev0 interface{}, p image.Point) bool {
	_ = sh.SeparatorHandle.OnInputEvent(ev0, p)
	if sh.Dragging {
		sh.col.resizeHandleWithSwapToPoint(&p)
	}
	switch ev := ev0.(type) {
	case *event.MouseDown:
		switch ev.Button {
		case event.ButtonWheelLeft:
			sh.col.resizeHandleWithSwapJump(true, &p)
		case event.ButtonWheelRight:
			sh.col.resizeHandleWithSwapJump(false, &p)
		}
	}
	return false
}
