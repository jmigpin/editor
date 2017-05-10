package ui

import (
	"image"
	"image/color"
	"image/draw"
	"log"

	"github.com/jmigpin/editor/drawutil"
	"github.com/jmigpin/editor/imageutil"
	"github.com/jmigpin/editor/uiutil"
	"github.com/jmigpin/editor/xgbutil"

	"github.com/BurntSushi/xgb/xproto"
)

const (
	ScrollbarWidth = 12
	SquareWidth    = ScrollbarWidth
	SeparatorWidth = 1
)

type UI struct {
	Win       *Window
	Layout    *Layout
	fface1    *drawutil.Face
	CursorMan *CursorMan
}

func NewUI(fface *drawutil.Face) (*UI, error) {
	ui := &UI{
		fface1: fface,
	}

	win, err := NewWindow()
	if err != nil {
		return nil, err
	}
	ui.Win = win

	// cursorman needs win in ui
	ui.CursorMan = NewCursorMan(ui)

	ui.Layout = NewLayout(ui)

	ui.Win.EvReg.Add(xproto.Expose,
		&xgbutil.ERCallback{ui.onExpose})
	ui.Win.EvReg.Add(xgbutil.QueueEmptyEventId,
		&xgbutil.ERCallback{ui.onQueueEmpty})

	return ui, nil
}
func (ui *UI) Close() {
	ui.Win.Close()
}
func (ui *UI) EventLoop() {
	ui.Win.RunEventLoop()
}
func (ui *UI) onExpose(ev0 xgbutil.EREvent) {
	ev := ev0.(xproto.ExposeEvent)

	// number of expose events to come
	if ev.Count > 0 {
		//// repaint just the exposed area
		//x0, y0 := int(ev.X), int(ev.Y)
		//x1, y1 := x0+int(ev.Width), y0+int(ev.Height)
		//r := image.Rect(x0, y0, x1, y1)
		//ui.PutImage(&r)

		return // wait for expose with count 0
	}

	err := ui.Win.UpdateImageSize()
	if err != nil {
		log.Println(err)
	} else {
		ib := ui.Win.Image().Bounds()
		if !ui.Layout.C.Bounds.Eq(ib) {
			ui.Layout.C.Bounds = ib
			ui.Layout.C.CalcChildsBounds()
		}
	}

	ui.Layout.C.NeedPaint()
}

func (ui *UI) onQueueEmpty(ev xgbutil.EREvent) {
	// paint after all events have been handled
	ui.Layout.C.PaintTreeIfNeeded(func(c *uiutil.Container) {
		// paint only the top container of the needed area
		ui.Win.PutImage(&c.Bounds)
	})
}

// Send paint request to the main event loop.
// Usefull for async methods that need to be painted.
func (ui *UI) RequestTreePaint() {
	ui.Win.EventLoop.EnqueueQEmptyEventIfConnQEmpty()
}

func (ui *UI) Image() draw.Image {
	return ui.Win.Image()
}
func (ui *UI) FillRectangle(r *image.Rectangle, c color.Color) {
	imageutil.FillRectangle(ui.Image(), r, c)
}

// Default fontface (used by textarea)
func (ui *UI) FontFace() *drawutil.Face {
	return ui.fface1
}

func (ui *UI) WarpPointer(p *image.Point) {
	ui.Win.WarpPointer(p)
}
func (ui *UI) WarpPointerToRectanglePad(r0 *image.Rectangle) {
	p, ok := ui.Win.QueryPointer()
	if !ok {
		return
	}
	// pad rectangle
	pad := 25
	r := *r0
	if r.Dx() < pad*2 {
		r.Min.X = r.Min.X + r.Dx()/2
		r.Max.X = r.Min.X
	} else {
		r.Min.X += pad
		r.Max.X -= pad
	}
	if r.Dy() < pad*2 {
		r.Min.Y = r.Min.Y + r.Dy()/2
		r.Max.Y = r.Min.Y
	} else {
		r.Min.Y += pad
		r.Max.Y -= pad
	}
	// put p inside
	if p.Y < r.Min.Y {
		p.Y = r.Min.Y
	} else if p.Y >= r.Max.Y {
		p.Y = r.Max.Y
	}
	if p.X < r.Min.X {
		p.X = r.Min.X
	} else if p.X >= r.Max.X {
		p.X = r.Max.X
	}
	ui.WarpPointer(p)
}
