package ui

import (
	"image"
	"image/draw"
	"log"
	"time"

	"golang.org/x/image/font"

	"github.com/jmigpin/editor/uiutil"
	"github.com/jmigpin/editor/uiutil/widget"
	"github.com/jmigpin/editor/xgbutil/evreg"
	"github.com/jmigpin/editor/xgbutil/xcursors"
	"github.com/jmigpin/editor/xgbutil/xinput"
	"github.com/jmigpin/editor/xgbutil/xwindow"

	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil/xcursor"
)

const (
	DrawFrameRate = 35
)

var (
	SeparatorWidth = 1
	ScrollbarWidth = 10
	SquareWidth    = 10
	ScrollbarLeft  = false
	ShadowsOn      = true
	ShadowMaxShade = 0.25
	ShadowSteps    = 8
)

func SetScrollbarAndSquareWidth(v int) {
	ScrollbarWidth = v
	SquareWidth = v
}

type UI struct {
	win    *xwindow.Window
	Layout Layout

	EvReg   *evreg.Register
	Events2 chan interface{}

	lastPaint       time.Time
	incompleteDraws int
}

func NewUI() (*UI, error) {
	ui := &UI{
		Events2: make(chan interface{}, 256),
	}

	ui.EvReg = evreg.NewRegister()
	ui.EvReg.Events = ui.Events2

	win, err := xwindow.NewWindow(ui.EvReg)
	if err != nil {
		return nil, err
	}
	win.SetWindowName("Editor")
	ui.win = win

	ui.Layout.Init(ui)

	ui.EvReg.Add(xproto.Expose, ui.onExpose)
	ui.EvReg.Add(evreg.ShmCompletionEventId, ui.onShmCompletion)
	ui.EvReg.Add(xinput.InputEventId, ui.onInput)
	ui.EvReg.Add(UIRunFuncEventId, ui.onRunFunc)

	return ui, nil
}
func (ui *UI) Close() {
	ui.win.Close()
}

func (ui *UI) onExpose(ev0 interface{}) {
	ui.UpdateImageSize()
	ui.Layout.MarkNeedsPaint()
}

func (ui *UI) UpdateImageSize() {
	err := ui.win.UpdateImageSize()
	if err != nil {
		log.Println(err)
	} else {
		ib := ui.win.Image().Bounds()
		if !ui.Layout.Bounds().Eq(ib) {
			ui.Layout.SetBounds(&ib)
			ui.Layout.CalcChildsBounds()
			ui.Layout.MarkNeedsPaint()
		}
	}
}

// This function should be called in the event loop after every event.
func (ui *UI) PaintIfNeeded() {
	const fps = DrawFrameRate
	now := time.Now()
	d := now.Sub(ui.lastPaint)
	canPaint := d > (time.Second / fps)
	if canPaint {
		painted := ui.paintIfNeeded2()
		if painted {
			//log.Printf("time since last paint %v", time.Now().Sub(ui.lastPaint))
			ui.lastPaint = now
		}
	} else {
		if len(ui.Events2) == 0 {
			// Didn't paint to avoid high fps. Need to ensure a new paint call will happen later.
			ui.EvReg.Enqueue(evreg.NoOpEventId, nil)
		}
	}
}

func (ui *UI) paintIfNeeded2() (painted bool) {
	// Still painting something else, don't paint now. This function should be called again uppon the draw completion event.
	if ui.incompleteDraws != 0 {
		return false
	}

	var u []*image.Rectangle
	widget.PaintIfNeeded(&ui.Layout, func(r *image.Rectangle) {
		painted = true
		u = append(u, r)
	})

	// union the rectangles into one put
	if len(u) > 0 {
		var r2 image.Rectangle
		for _, r := range u {
			r2 = r2.Union(*r)
		}
		ui.putImage(&r2)
	}

	return painted
}

func (ui *UI) putImage(r *image.Rectangle) {
	ui.incompleteDraws++
	ui.win.PutImage(r)
}
func (ui *UI) onShmCompletion(_ interface{}) {
	ui.incompleteDraws--
}

func (ui *UI) onInput(ev0 interface{}) {
	ev := ev0.(*xinput.InputEvent)
	uiutil.ApplyInputEventInBounds(&ui.Layout, ev.Event, ev.Point)
}

func (ui *UI) RequestPaint() {
	ui.EvReg.Enqueue(evreg.NoOpEventId, nil)
}

// Implements widget.Context
func (ui *UI) Image() draw.Image {
	return ui.win.Image()
}

// Implements widget.Context
func (ui *UI) FontFace1() font.Face {
	return FontFace
}

// Implements widget.Context
func (ui *UI) SetCursor(c widget.Cursor) {
	sc := ui.win.Cursors.SetCursor
	switch c {
	case widget.NoCursor:
		sc(xcursors.XCNone)
	case widget.DefaultCursor:
		sc(xcursors.XCNone)
	case widget.NSResizeCursor:
		sc(xcursor.SBVDoubleArrow)
	case widget.WEResizeCursor:
		sc(xcursor.SBHDoubleArrow)
	case widget.CloseCursor:
		sc(xcursor.XCursor)
	case widget.MoveCursor:
		sc(xcursor.Fleur)
	case widget.PointerCursor:
		sc(xcursor.Hand2)
	case widget.TextCursor:
		sc(xcursor.XTerm)
	}
}

func (ui *UI) QueryPointer() (*image.Point, bool) {
	return ui.win.QueryPointer()
}
func (ui *UI) WarpPointer(p *image.Point) {
	ui.win.WarpPointer(p)
	uiutil.InputEventWarpedPointUntilMouseMove(*p)
}

func (ui *UI) WarpPointerToRectanglePad(r0 *image.Rectangle) {
	p, ok := ui.QueryPointer()
	if !ok {
		return
	}

	pad := 5

	set := func(v *int, min, max int) {
		if max-min < pad*2 {
			*v = min + (max-min)/2
		} else {
			if *v < min+pad {
				*v = min + pad
			} else if *v > max-pad {
				*v = max - pad
			}
		}
	}

	r := *r0
	set(&p.X, r.Min.X, r.Max.X)
	set(&p.Y, r.Min.Y, r.Max.Y)

	ui.WarpPointer(p)
}

func (ui *UI) RequestPrimaryPaste() (string, error) {
	return ui.win.Paste.RequestPrimary()
}
func (ui *UI) RequestClipboardPaste() (string, error) {
	return ui.win.Paste.RequestClipboard()
}
func (ui *UI) SetClipboardCopy(v string) {
	ui.win.Copy.SetClipboard(v)
}
func (ui *UI) SetPrimaryCopy(v string) {
	ui.win.Copy.SetPrimary(v)
}

func (ui *UI) EnqueueRunFunc(f func()) {
	ev := &UIRunFuncEvent{f}
	ui.EvReg.Enqueue(UIRunFuncEventId, ev)
}
func (ui *UI) onRunFunc(ev0 interface{}) {
	ev := ev0.(*UIRunFuncEvent)
	ev.F()
}

const (
	UIRunFuncEventId = evreg.UIEventIdStart + iota
)

type UIRunFuncEvent struct {
	F func()
}
