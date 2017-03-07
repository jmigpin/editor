package ui

import (
	"fmt"
	"image"
	"image/draw"

	"github.com/jmigpin/editor/drawutil"
	"github.com/jmigpin/editor/xutil"
	"github.com/jmigpin/editor/xutil/dragndrop"
	"github.com/jmigpin/editor/xutil/keybmap"

	"github.com/BurntSushi/xgb/xproto"
)

const (
	ScrollbarWidth = 12
	SeparatorWidth = 1
)

type UI struct {
	Win    *xutil.Window
	Layout *Layout
	fface1 *drawutil.Face

	events chan Event

	OnEvent func(Event)
}

type Event interface{}
type EmptyEvent struct{}
type KeyPressEvent struct {
	Key *keybmap.Key
}
type ButtonPressEvent struct {
	Button *keybmap.Button
}
type ButtonReleaseEvent struct {
	Button *keybmap.Button
}
type MotionNotifyEvent struct {
	Modifiers keybmap.Modifiers
}
type TextAreaCmdEvent struct {
	TextArea *TextArea
}
type TextAreaSetTextEvent struct {
	TextArea *TextArea
	OldArea  image.Rectangle
}
type TextAreaSetOffsetYEvent struct {
	TextArea *TextArea
}
type TextAreaScrollEvent struct {
	TextArea *TextArea
	Up       bool
}
type SquareButtonReleaseEvent struct {
	Square *Square
	Button *keybmap.Button
	Point  *image.Point
}
type SquareRootButtonReleaseEvent struct {
	Square *Square
	Button *keybmap.Button
	Point  *image.Point
}
type SquareRootMotionNotifyEvent struct {
	Square    *Square
	Modifiers keybmap.Modifiers
	Point     *image.Point
}
type RowKeyPressEvent struct {
	Row *Row
	Key *keybmap.Key
}
type RowCloseEvent struct {
	Row *Row
}
type ColumnDndPositionEvent struct {
	Event  *dragndrop.PositionEvent
	Point  *image.Point
	Column *Column
}
type ColumnDndDropEvent struct {
	Event  *dragndrop.DropEvent
	Point  *image.Point
	Column *Column
}

func NewUI(fface *drawutil.Face) (*UI, error) {
	ui := &UI{fface1: fface}

	win, err := xutil.NewWindow()
	if err != nil {
		return nil, err
	}
	ui.Win = win

	ui.Layout = NewLayout(ui)
	return ui, nil
}
func (ui *UI) Close() {
	if ui.events != nil {
		close(ui.events)
	}
	ui.Win.Close()
}
func (ui *UI) EventLoop() {
	ui.events = make(chan Event)
	go func() {
		for {
			ev, ok := ui.Win.WaitForEvent()
			if !ok { // conn closed
				break
			}
			ui.events <- ev
		}
	}()
	for {
		var ev xutil.Event
		var ok bool
		select {
		case ev, ok = <-ui.events: // non-block
		default:
			// paint after all events have been handled
			ui.Layout.TreePaint()

			ev, ok = <-ui.events // block
		}
		if !ok {
			break
		}
		ui.onXUtilEvent(ev)
	}
}

// Usefull for NeedPaint() calls made inside a goroutine that have no way  of requesting a paint later since the event loop only paints after all events have been handled, so it doesn't paint if there are no events (hence using an empty event).
func (ui *UI) RequestTreePaint() {
	ui.events <- &EmptyEvent{}
}

func (ui *UI) onXUtilEvent(ev xutil.Event) {
	// TODO: receive shm.CompletionEvent, and count the waits before drawing next (performance)

	switch ev0 := ev.(type) {
	case error:
		ui.OnEvent(ev0) // pass error to handler
	case *EmptyEvent:
		// do nothing, it will trigger the event loop to loop
	//case *xutil.ConnClosedEvent:
	//fmt.Println("x connection closed")
	case xproto.ExposeEvent:
		if ev0.Count > 0 {
			return // wait for expose with count 0
		}
		ui.adjustRootImageSize()
		ui.Layout.NeedPaint()
	case xproto.KeyPressEvent:
		p := &image.Point{int(ev0.EventX), int(ev0.EventY)}
		k := ui.Win.KeybMap.NewKey(ev0.Detail, ev0.State)
		ev2 := &KeyPressEvent{k}
		ui.Layout.pointEvent(p, ev2)
	case xproto.KeyReleaseEvent:
		// didn't registered to receive, but still showing up
	case xproto.ButtonPressEvent:
		p := &image.Point{int(ev0.EventX), int(ev0.EventY)}
		b := ui.Win.KeybMap.NewButton(ev0.Detail, ev0.State)
		ev2 := &ButtonPressEvent{b}
		ui.Layout.pointEvent(p, ev2)
	case xproto.ButtonReleaseEvent:
		p := &image.Point{int(ev0.EventX), int(ev0.EventY)}
		b := ui.Win.KeybMap.NewButton(ev0.Detail, ev0.State)
		ev2 := &ButtonReleaseEvent{b}
		ui.Layout.pointEvent(p, ev2)
	case xproto.MotionNotifyEvent:
		p := &image.Point{int(ev0.EventX), int(ev0.EventY)}
		m := ui.Win.KeybMap.NewModifiers(ev0.State)
		ev2 := &MotionNotifyEvent{m}
		ui.Layout.pointEvent(p, ev2)
	case *dragndrop.PositionEvent:
		p, err := ev0.WindowPoint()
		if err != nil {
			ui.OnEvent(err)
			return
		}
		ui.Layout.pointEvent(p, ev0)
		// dnd position must receive a reply, make one if no one handled it
		if !ev0.Replied {
			ev0.ReplyDeny()
		}
	case *dragndrop.DropEvent:
		p, err := ev0.WindowPoint()
		if err != nil {
			ui.OnEvent(err)
			return
		}
		ui.Layout.pointEvent(p, ev0)
		// dnd position must receive a reply, make one if no one handled it
		if !ev0.Replied {
			ev0.ReplyDeny()
		}
	default:
		ev2 := fmt.Errorf("ui unhandled event: %v", ev)
		ui.OnEvent(ev2)
	}
}
func (ui *UI) PushEvent(ev Event) {
	ui.onUIPushedEvent(ev) // no queue, run directly
}
func (ui *UI) onUIPushedEvent(ev Event) {
	switch ev0 := ev.(type) {
	case *SquareButtonReleaseEvent:
		switch t0 := ev0.Square.Data.(type) {
		case *Row:
			ui.onRowSquareButtonRelease(t0, ev0)
		case *Column:
			ui.onColumnSquareButtonRelease(t0, ev0)
		}
	case *SquareRootButtonReleaseEvent:
		switch t0 := ev0.Square.Data.(type) {
		case *Row:
			ui.onRowSquareRootButtonRelease(t0, ev0)
		case *Column:
			ui.onColumnSquareRootButtonRelease(t0, ev0)
		}
	case *SquareRootMotionNotifyEvent:
		switch t0 := ev0.Square.Data.(type) {
		case *Row:
			ui.onRowSquareRootMotionNotify(t0, ev0)
		case *Column:
			ui.onColumnSquareRootMotionNotify(t0, ev0)
		}
	case *TextAreaScrollEvent:
		switch t0 := ev0.TextArea.Data.(type) {
		case *Row:
			if ev0.TextArea == t0.TextArea {
				t0.scrollbar.NeedPaint()
			}
		}
	case *TextAreaSetTextEvent:
		ui.onTextAreaSetText(ev0)
		// also pass to next handler
		ui.OnEvent(ev0)
	case *TextAreaSetOffsetYEvent:
		switch t0 := ev0.TextArea.Data.(type) {
		case *Row:
			if ev0.TextArea == t0.TextArea {
				t0.scrollbar.CalcOwnArea()
				t0.scrollbar.NeedPaint()
			}
		}
	default:
		ui.OnEvent(ev)
	}
}
func (ui *UI) onTextAreaSetText(ev0 *TextAreaSetTextEvent) {
	ta := ev0.TextArea
	switch t0 := ev0.TextArea.Data.(type) {
	case *Toolbar:
		// update toolbar parent container
		switch t1 := t0.Data.(type) {
		case *Layout:
			t1.CalcOwnArea()
			t1.NeedPaint()
		case *Row:
			t1.CalcOwnArea()
			t1.NeedPaint()
		}
		// keep pointer inside the area if it was in before
		p, ok := ta.UI.Win.QueryPointer()
		wasIn := ok && p.In(ev0.OldArea)
		if wasIn {
			ta.UI.WarpPointerToRectangle(&ta.Area)
		}
	case *Row:
		// update scrollbar
		if ta == t0.TextArea {
			t0.scrollbar.CalcOwnArea()
			t0.scrollbar.NeedPaint()
		}
	}
}
func (ui *UI) onRowSquareButtonRelease(row *Row, ev *SquareButtonReleaseEvent) {
	if ev.Button.Button2() {
		row.Close()
	}
}
func (ui *UI) onRowSquareRootButtonRelease(row *Row, ev *SquareRootButtonReleaseEvent) {
	if ev.Button.Button1() {
		col := row.Col
		if ev.Button.Mods.Control() {
			col.Cols.MoveColumnToPoint(col, ev.Point)
		} else {
			c, i, ok := col.Cols.PointRowPosition(row, ev.Point)
			if ok {
				col.Cols.MoveRowToColumn(row, c, i)
			}
		}
	}
}
func (ui *UI) onColumnSquareButtonRelease(col *Column, ev *SquareButtonReleaseEvent) {
	if ev.Button.Button2() {
		col.Cols.RemoveColumnEnsureOne(col)
	}
}
func (ui *UI) onColumnSquareRootButtonRelease(col *Column, ev *SquareRootButtonReleaseEvent) {
	if ev.Button.Button1() {
		col.Cols.MoveColumnToPoint(col, ev.Point)
	}
}
func (ui *UI) onRowSquareRootMotionNotify(row *Row, ev *SquareRootMotionNotifyEvent) {
	if ev.Modifiers.Button3() {
		p2 := ev.Point.Add(row.Square.PressPointPad)
		col := row.Col
		col.Cols.resizeColumn(col, p2.X)
	}
}
func (ui *UI) onColumnSquareRootMotionNotify(col *Column, ev *SquareRootMotionNotifyEvent) {
	if ev.Modifiers.Button3() {
		p2 := ev.Point.Add(col.Square.PressPointPad)
		col.Cols.resizeColumn(col, p2.X)
	}
}

func (ui *UI) adjustRootImageSize() {
	wgeom, err := ui.Win.GetGeometry()
	if err != nil {
		fmt.Println(err)
		return
	}
	w := int(wgeom.Width)
	h := int(wgeom.Height)

	// make new image
	r := image.Rect(0, 0, w, h)
	if !r.Eq(ui.Layout.Area) {
		if err := ui.Win.ShmWrap.NewImage(&r); err != nil {
			fmt.Println(err)
			return
		}
		ui.Layout.Area = r
		ui.Layout.CalcOwnArea()
	}
}

// Provides image to draw for drawutil (ex: fillrectangle).
func (ui *UI) RootImage() draw.Image {
	return ui.Win.ShmWrap.Image()
}

// Provides image to draw for drawutil (ex: textarea).
func (ui *UI) RootImageSubImage(r *image.Rectangle) draw.Image {
	return ui.Win.ShmWrap.Image().SubImage(*r)
}

func (ui *UI) PutRootImage(rect *image.Rectangle) {
	ui.Win.ShmWrap.PutImage(ui.Win.GCtx, rect)
}

// Default fontface (used by textarea)
func (ui *UI) FontFace() *drawutil.Face {
	return ui.fface1
}

// Should be called when a button is pressed and need the motion-notify-events to keep coming since the program expects only pointer-motion-hints.
func (ui *UI) RequestMotionNotify() {
	ui.Win.RequestMotionNotify()
}

func (ui *UI) WarpPointer(p *image.Point) {
	ui.Win.WarpPointer(p)
}
func (ui *UI) WarpPointerToRectangle(r *image.Rectangle) {
	p, ok := ui.Win.QueryPointer()
	if !ok {
		return
	}
	if p.In(*r) {
		return
	}
	pad := 3
	if p.Y < r.Min.Y {
		p.Y = r.Min.Y + pad
	} else if p.Y >= r.Max.Y {
		p.Y = r.Max.Y - pad
	}
	if p.X < r.Min.X {
		p.X = r.Min.X + pad
	} else if p.X >= r.Max.X {
		p.X = r.Max.X - pad
	}
	ui.WarpPointer(p)
}
