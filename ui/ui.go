package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/jmigpin/editor/drawutil"
	"github.com/jmigpin/editor/imageutil"
	"github.com/jmigpin/editor/uiutil"
	"github.com/jmigpin/editor/xutil"
	"github.com/jmigpin/editor/xutil/xgbutil"

	"github.com/BurntSushi/xgb/xproto"
)

const (
	ScrollbarWidth = 12
	SquareWidth    = ScrollbarWidth
	SeparatorWidth = 1
)

type UI struct {
	Win    *xutil.Window
	Layout *Layout
	fface1 *drawutil.Face
}

func NewUI(fface *drawutil.Face) (*UI, error) {
	ui := &UI{fface1: fface}

	win, err := xutil.NewWindow()
	if err != nil {
		return nil, err
	}
	ui.Win = win

	ui.Layout = NewLayout(ui)

	ui.Win.EvReg.Add(xproto.Expose,
		&xgbutil.ERCallback{ui.onExposeEvent})
	ui.Win.EvReg.Add(xgbutil.QueueEmptyEventId,
		&xgbutil.ERCallback{ui.onQueueEmptyEvent})

	return ui, nil
}
func (ui *UI) Close() {
	ui.Win.Close()
}
func (ui *UI) EventLoop() {
	ui.Win.EventLoop()
}
func (ui *UI) onExposeEvent(ev0 xgbutil.EREvent) {
	ev := ev0.(xproto.ExposeEvent)

	//if ev.Count > 0 { // number of expose event to come
	//return // wait for expose with count 0
	//}

	r := ui.winGeometry()
	if !r.Eq(ui.Layout.C.Bounds) {
		// new image
		if err := ui.Win.ShmWrap.NewImage(&r); err != nil {
			fmt.Println(err)
			return
		}
		ui.Layout.C.Bounds = r
		ui.Layout.C.CalcChildsBounds()
		ui.Layout.C.NeedPaint()
	} else {
		// repaint just the exposed area
		x0, y0 := int(ev.X), int(ev.Y)
		x1, y1 := x0+int(ev.Width), y0+int(ev.Height)
		r := image.Rect(x0, y0, x1, y1)
		ui.PutImage(&r)
	}
}
func (ui *UI) winGeometry() *image.Rectangle {
	wgeom, err := ui.Win.GetGeometry()
	if err != nil {
		fmt.Println(err)
		return
	}
	w := int(wgeom.Width)
	h := int(wgeom.Height)
	return &image.Rect(0, 0, w, h)
}
func (ui *UI) onQueueEmptyEvent(ev xgbutil.EREvent) {
	// paint after all events have been handled
	ui.Layout.C.PaintTreeIfNeeded(func(c *uiutil.Container) {
		// paint only the top container of the needed area
		ui.PutImage(&c.Bounds)
	})
}

// Usefull for NeedPaint() calls made inside a goroutine that have no way  of requesting a paint later since the event loop only paints after all events have been handled, so it doesn't paint if there are no events (hence using an empty event).
func (ui *UI) RequestTreePaint() {
	ui.Win.EvReg.Emit(xgbutil.QueueEmptyEventId, nil)
}

func (ui *UI) Image() draw.Image {
	return ui.Win.ShmWrap.Image()
}
func (ui *UI) FillRectangle(r *image.Rectangle, c color.Color) {
	imageutil.FillRectangle(ui.Image(), r, c)
}
func (ui *UI) PutImage(rect *image.Rectangle) {
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
	// put p inside
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
