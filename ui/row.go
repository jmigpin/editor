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
		//resizeCol := ev.Button.Mods.HasControl()
		//row.startResizeToPoint(ev.Point)
		row.startRowResizeToPoint(ev.Point)
	case ev.Button.Button(2):
		// indicate close
		ui.CursorMan.SetCursor(xcursor.XCursor)
	case ev.Button.Button(3):
		row.startColumnResizeToPoint(ev.Point)
	case ev.Button.Button(4):
		row.resizeWithPush(true)
	case ev.Button.Button(5):
		row.resizeWithPush(false)
	}
}
func (row *Row) onSquareMotionNotify(ev0 interface{}) {
	ev := ev0.(*SquareMotionNotifyEvent)
	switch {
	case ev.Mods.HasButton(1):
		row.detectAndResizeToPoint(ev.Point)
	case ev.Mods.IsButton(3):
		row.detectAndResizeToPoint(ev.Point)
	}
}
func (row *Row) onSquareButtonRelease(ev0 interface{}) {
	ui := row.Col.Cols.Layout.UI
	ui.CursorMan.UnsetCursor()

	ev := ev0.(*SquareButtonReleaseEvent)
	switch {
	case ev.Button.Mods.HasButton(1):
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

func (row *Row) startColumnResizeToPoint(p *image.Point) {
	row.resize.detect = false
	row.resize.on = true
	row.resize.origin = p.Sub(row.Square.Bounds().Min)
	if !ScrollbarLeft {
		row.resize.origin.X = p.Sub(row.Square.Bounds().Max).X
	}

	ui := row.Col.Cols.Layout.UI
	ui.CursorMan.SetCursor(xcursor.SBHDoubleArrow)
	row.resize.typ = ResizeColumnRType

	row.resizeToPoint(p)
}

func (row *Row) startRowResizeToPoint(p *image.Point) {
	row.resize.detect = false
	row.resize.on = true
	row.resize.origin = p.Sub(row.Square.Bounds().Min)
	if !ScrollbarLeft {
		row.resize.origin.X = p.Sub(row.Square.Bounds().Max).X
	}

	ui := row.Col.Cols.Layout.UI
	ui.CursorMan.SetCursor(xcursor.Fleur)
	row.resize.typ = ResizeRowRType

	row.resizeToPoint(p)
}

func (row *Row) startResizeToPoint(p *image.Point) {
	row.resize.detect = true
	row.resize.on = false
	row.resize.origin = p.Sub(row.Square.Bounds().Min)
	if !ScrollbarLeft {
		row.resize.origin.X = p.Sub(row.Square.Bounds().Max).X
	}
}
func (row *Row) detectAndResizeToPoint(p *image.Point) {
	if row.resize.detect {
		row.detectResize(p)
	}
	row.resizeToPoint(p)
}

func (row *Row) resizeToPoint(p *image.Point) {
	if row.resize.on {
		switch row.resize.typ {
		case ResizeRowRType:
			row.resizeRowToPoint(p)
		case ResizeColumnRType:
			row.resizeColumnToPoint(p)
		default:
			panic("!")
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
		default:
			panic("!")
		}
	}
}

func (row *Row) detectResize(p *image.Point) {
	u := p.Sub(row.Square.Bounds().Min)
	if !ScrollbarLeft {
		u.X = p.Sub(row.Square.Bounds().Max).X
	}
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
	if a <= 15 {
		// horizontal
		sc(xcursor.SBHDoubleArrow)
		row.resize.typ = ResizeColumnRType
	} else {
		// any other angle
		sc(xcursor.Fleur)
		row.resize.typ = ResizeRowRType
	}

	//// re-keep origin to avoid jump
	//// difficult to push beyond other rows if the square has big Y
	//row.resize.origin = p.Sub(row.Square.Bounds().Min)
	//if !ScrollbarLeft {
	//	row.resize.origin.X = p.Sub(row.Square.Bounds().Max).X
	//}

	// accurate position (makes jump)
	// works best as well for accurate swaps
	row.resize.origin = image.Point{}

	//// reposition pointer to look accurate without jumping the movement
	//p2 := row.Square.Bounds().Min.Add(row.resize.origin)
	//if !ScrollbarLeft {
	//	p2.X = row.Square.Bounds().Max.Add(row.resize.origin).X
	//}
	//row.Col.Cols.Layout.UI.WarpPointer(&p2)

	row.resize.detect = false
	row.resize.on = true
}

func (row *Row) resizeRowToPoint(p *image.Point) {
	col, ok := row.Col.Cols.PointColumn(p)
	if !ok {
		return
	}
	if col != row.Col {
		// move to another column
		next, ok := col.PointRow(p)
		if ok {
			next, _ = next.NextRow()
		}
		if next != row {
			row.Col.removeRow(row)
			col.insertBefore(row, next)
		}

		//// take the opportunity and make the origin accurate
		//row.resize.origin = image.Point{}
	}

	bounds := row.Col.Bounds()
	dy := float64(bounds.Dy())
	perc := float64(p.Sub(row.resize.origin).Sub(bounds.Min).Y) / dy
	min := 30 / dy

	percIsTop := true
	rl := row.Col.rowsLayout
	rl.ResizeEndPercentWithSwap(rl, row, perc, percIsTop, min)

	row.Col.CalcChildsBounds()
	row.Col.MarkNeedsPaint()
}
func (row *Row) resizeColumnToPoint(p *image.Point) {
	row.Col.resizeToPointOrigin(p, &row.resize.origin)
}

func (row *Row) maximizeRow() {
	col := row.Col
	dy := float64(col.Bounds().Dy())
	min := 30 / dy
	col.rowsLayout.MaximizeEndPercentNode(row, min)
	col.CalcChildsBounds()
	col.MarkNeedsPaint()
}

func (row *Row) resizeWithPush(up bool) {
	col := row.Col
	dy := float64(col.Bounds().Dy())
	min := 30 / dy

	jump := 30
	if up {
		jump *= -1
	}
	perc := float64(row.Bounds().Min.Y-col.Bounds().Min.Y+jump) / dy

	percIsTop := true
	col.rowsLayout.ResizeEndPercentWithPush(row, perc, percIsTop, min)

	col.CalcChildsBounds()
	col.MarkNeedsPaint()

	// keep pointer inside the square (newly calculated)
	b := row.Square.Bounds()
	sqCenter := b.Min.Add(b.Max.Sub(b.Min).Div(2))
	row.Col.Cols.Layout.UI.WarpPointer(&sqCenter)
}

func (row *Row) ResizeTextAreaIfVerySmall() {
	col := row.Col
	dy := float64(col.Bounds().Dy())
	min := 30 / dy
	ta := row.TextArea
	taMin := ta.LineHeight().Ceil()

	taDy := ta.Bounds().Dy()
	if taDy > taMin {
		return
	}

	hint := image.Point{row.Bounds().Dx(), 1000000}
	rm := row.Measure(hint)
	tm := row.TextArea.Measure(hint)
	size := (rm.Y - tm.Y) + taMin

	// push siblings down
	perc := float64(row.Bounds().Min.Sub(col.Bounds().Min).Y+size) / dy
	percIsTop := false
	col.rowsLayout.ResizeEndPercentWithPush(row, perc, percIsTop, min)

	col.CalcChildsBounds()
	col.MarkNeedsPaint()

	// check if good already
	taDy = ta.Bounds().Dy()
	if taDy > taMin {
		return
	}

	// push siblings up
	perc = float64(row.Bounds().Max.Sub(col.Bounds().Min).Y-size) / dy
	percIsTop = true
	col.rowsLayout.ResizeEndPercentWithPush(row, perc, percIsTop, min)

	col.CalcChildsBounds()
	col.MarkNeedsPaint()
}

type RowRType int

const (
	ResizeRowRType RowRType = iota
	ResizeColumnRType
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
