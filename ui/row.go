package ui

import (
	"image"
	"math"

	"github.com/BurntSushi/xgbutil/xcursor"
	"github.com/jmigpin/editor/uiutil/widget"
	"github.com/jmigpin/editor/xgbutil/evreg"
	"github.com/jmigpin/editor/xgbutil/xinput"
)

type Row struct {
	widget.FlowLayout
	Square     *Square
	Toolbar    *Toolbar
	scrollArea *ScrollArea
	TextArea   *TextArea
	sep        *widget.Space

	Col   *Column
	EvReg *evreg.Register

	buttonPressed bool

	resize struct {
		detect bool
		on     bool
		origin image.Point
		typ    RowRType
	}
}

func NewRow(col *Column) *Row {
	row := &Row{Col: col}

	ui := row.Col.Cols.Layout.UI

	row.EvReg = evreg.NewRegister()

	row.Toolbar = NewToolbar(ui, row)
	row.Toolbar.SetExpand(true, false)

	row.Square = NewSquare(ui)
	row.Square.SetFill(false, true)
	row.Square.EvReg.Add(SquareButtonPressEventId, row.onSquareButtonPress)
	row.Square.EvReg.Add(SquareButtonReleaseEventId, row.onSquareButtonRelease)
	row.Square.EvReg.Add(SquareMotionNotifyEventId, row.onSquareMotionNotify)

	// row separator from other rows
	row.sep = widget.NewSpace(ui)
	row.sep.SetExpand(true, false)
	row.sep.Size.Y = SeparatorWidth
	row.sep.Color = SeparatorColor

	// square and toolbar
	tb := &widget.FlowLayout{}
	sep1 := widget.NewSpace(ui)
	sep1.Color = RowInnerSeparatorColor
	sep1.Size.X = SeparatorWidth
	sep1.SetFill(false, true)
	if ScrollbarLeft {
		widget.AppendChilds(tb, row.Square, sep1, row.Toolbar)
	} else {
		widget.AppendChilds(tb, row.Toolbar, sep1, row.Square)
	}

	// toolbar separator from scrollarea
	tbSep := widget.NewSpace(ui)
	tbSep.SetExpand(true, false)
	tbSep.Size.Y = SeparatorWidth
	tbSep.Color = RowInnerSeparatorColor

	// scrollarea with textarea
	row.TextArea = NewTextArea(ui)
	row.TextArea.Colors = &TextAreaColors
	row.scrollArea = NewScrollArea(ui, row.TextArea)
	row.scrollArea.SetExpand(true, true)
	row.scrollArea.LeftScroll = ScrollbarLeft
	row.scrollArea.ScrollWidth = ScrollbarWidth
	row.scrollArea.Fg = ScrollbarFgColor
	row.scrollArea.Bg = ScrollbarBgColor

	row.YAxis = true
	widget.AppendChilds(row, row.sep, tb, tbSep, row.scrollArea)

	return row
}
func (row *Row) activate() {
	// deactivate previous active row
	for _, c := range row.Col.Cols.Columns() {
		for _, r := range c.Rows() {
			r.Square.SetValue(SquareActive, false)
		}
	}
	// activate this row
	row.Square.SetValue(SquareActive, true)
}
func (row *Row) Close() {
	row.Col.removeRow(row)
	row.EvReg.RunCallbacks(RowCloseEventId, &RowCloseEvent{row})
}
func (row *Row) onSquareButtonPress(ev0 interface{}) {
	ev := ev0.(*SquareButtonPressEvent)
	ui := row.Col.Cols.Layout.UI
	switch {
	case ev.Button.Button(1):
		// indicate moving
		//ui.CursorMan.SetCursor(xcursor.Fleur)
		row.startResizeToPoint(ev.Point)
	case ev.Button.Button(2):
		// indicate close
		ui.CursorMan.SetCursor(xcursor.XCursor)
	}
}
func (row *Row) onSquareMotionNotify(ev0 interface{}) {
	ev := ev0.(*SquareMotionNotifyEvent)
	switch {
	case ev.Mods.IsButton(1):
		row.detectAndResizeToPoint(ev.Point)
	case ev.Mods.IsButton(3):
	}
}
func (row *Row) onSquareButtonRelease(ev0 interface{}) {
	ui := row.Col.Cols.Layout.UI
	ui.CursorMan.UnsetCursor()

	ev := ev0.(*SquareButtonReleaseEvent)
	switch {
	case ev.Button.Mods.IsButton(1):
		if !row.resize.on {
			if ev.Point.In(row.Square.Bounds()) {
				row.maximizeRow()
			}
		} else {
			row.endResizeToPoint(ev.Point)
		}
	case ev.Button.Mods.IsButton(2):
		if ev.Point.In(row.Square.Bounds()) {
			row.Close()
		}
	}
}

func (row *Row) OnInputEvent(ev0 interface{}, p image.Point) bool {
	switch evt := ev0.(type) {
	case *xinput.KeyPressEvent:
		row.onKeyPress(evt)
	case *xinput.ButtonPressEvent:
		row.onButtonPress(evt)
	case *xinput.ButtonReleaseEvent:
		row.onButtonRelease(evt)
	}
	return false
}

func (row *Row) onKeyPress(ev *xinput.KeyPressEvent) {
	row.activate()
	ev2 := &RowKeyPressEvent{row, ev.Key}
	row.EvReg.RunCallbacks(RowKeyPressEventId, ev2)
}
func (row *Row) onButtonPress(ev *xinput.ButtonPressEvent) {
	row.buttonPressed = true
}
func (row *Row) onButtonRelease(ev *xinput.ButtonReleaseEvent) {
	if !row.buttonPressed {
		return
	}
	row.buttonPressed = false
	row.activate()
}
func (row *Row) WarpPointer() {
	row.Square.WarpPointer()
}

func (row *Row) NextRow() (*Row, bool) {
	u := row.Next()
	if u == nil {
		return nil, false
	}
	return u.(*Row), true
}

func (row *Row) HideSeparator(v bool) {
	if row.sep.Hidden() != v {
		row.sep.SetHidden(v)
		row.MarkNeedsPaint()
	}
}

func (row *Row) startResizeToPoint(p *image.Point) {
	row.resize.detect = true
	row.resize.on = false
	row.resize.origin = p.Sub(row.Square.Bounds().Min)
}
func (row *Row) detectAndResizeToPoint(p *image.Point) {
	if row.resize.detect {
		row.detectResize(p)
	}
	if row.resize.on {
		switch row.resize.typ {
		case ResizeRowRType:
			row.resizeRowToPoint(p)
		case ResizeColumnRType:
			row.resizeColumnToPoint(p)
		case MoveRowRType:
		}
	}
}
func (row *Row) endResizeToPoint(p *image.Point) {
	if row.resize.on {
		row.resize.on = false
		switch row.resize.typ {
		case ResizeRowRType:
			row.resizeRowToPoint(p)
		case ResizeColumnRType:
			row.resizeColumnToPoint(p)
		case MoveRowRType:
			row.moveRowToPoint(p)
		}
	}
}

func (row *Row) detectResize(p *image.Point) {
	u := p.Sub(row.Square.Bounds().Min)
	w := u.Sub(row.resize.origin)
	x := math.Abs(float64(w.X))
	y := math.Abs(float64(w.Y))

	// give some pixels to make the decision
	dist := math.Sqrt(x*x + y*y)
	if dist < 15 {
		return
	}

	// detect
	a := math.Atan(y/x) * 180.0 / math.Pi
	sc := row.Col.Cols.Layout.UI.CursorMan.SetCursor
	if a <= 25 {
		// horizontal
		sc(xcursor.SBHDoubleArrow)
		row.resize.typ = ResizeColumnRType
	} else if a <= 75 {
		// diagonal
		sc(xcursor.Fleur)
		row.resize.typ = MoveRowRType
	} else {
		// vertical
		sc(xcursor.SBVDoubleArrow)
		row.resize.typ = ResizeRowRType
	}

	// re-keep origin to avoid jump
	row.resize.origin = p.Sub(row.Square.Bounds().Min)

	row.resize.detect = false
	row.resize.on = true
}

func (row *Row) resizeRowToPoint(p *image.Point) {
	bounds := row.Col.Bounds()
	dy := float64(bounds.Dy())
	perc := float64(p.Sub(row.resize.origin).Sub(bounds.Min).Y) / dy
	min := 30 / dy

	row.Col.rowsLayout.ResizeEndPercent(row, perc, min)
	row.Col.rowsLayout.AttemptToSwap(row.Col.rowsLayout, row, perc, min)

	row.Col.fixFirstRowSeparatorAndSquare()
	row.Col.CalcChildsBounds()
	row.Col.MarkNeedsPaint()
}
func (row *Row) resizeColumnToPoint(p *image.Point) {
	row.Col.resizeToPointOrigin(p, &row.resize.origin)
}

func (row *Row) moveRowToPoint(p *image.Point) {
	col, ok := row.Col.Cols.PointColumn(p)
	if !ok {
		return
	}
	next, ok := col.PointRow(p)
	if ok {
		next, _ = next.NextRow()
	}
	if next != row {
		row.Col.removeRow(row)
		col.insertBefore(row, next)
	}
	row.resize.origin = image.Point{} // accurate drop
	row.resizeRowToPoint(p)
	row.WarpPointer()
}
func (row *Row) maximizeRow() {
	col := row.Col
	dy := float64(col.Bounds().Dy())
	min := 30 / dy
	col.rowsLayout.MaximizeEndPercentNode(row, min)
	col.CalcChildsBounds()
	col.MarkNeedsPaint()
}

type RowRType int

const (
	ResizeRowRType RowRType = iota
	ResizeColumnRType
	MoveRowRType
)

const (
	RowKeyPressEventId = iota
	RowCloseEventId
)

type RowKeyPressEvent struct {
	Row *Row
	Key *xinput.Key
}
type RowCloseEvent struct {
	Row *Row
}
