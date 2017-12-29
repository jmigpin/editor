package ui

import (
	"image"
	"image/draw"
	"log"
	"time"

	"golang.org/x/image/font"

	"github.com/jmigpin/editor/driver"
	"github.com/jmigpin/editor/uiutil"
	"github.com/jmigpin/editor/uiutil/event"
	"github.com/jmigpin/editor/uiutil/widget"
)

const (
	DrawFrameRate = 35
	FlashDuration = 500 * time.Millisecond
)

var (
	SeparatorWidth = 1
	ScrollbarWidth = 10
	ScrollbarLeft  = false
	ShadowsOn      = true
	ShadowMaxShade = 0.25
	ShadowSteps    = 8
)

type UI struct {
	Layout          Layout
	AfterInputEvent func(ev interface{}, p image.Point)
	OnError         func(error)

	win             driver.Window
	events          chan<- interface{}
	lastPaint       time.Time
	incompleteDraws int
	curCursor       widget.Cursor
}

func NewUI(events chan<- interface{}, winName string) (*UI, error) {
	win, err := driver.NewWindow()
	if err != nil {
		return nil, err
	}
	win.SetWindowName(winName)

	// start window event loop with mousemove event filter
	events2 := make(chan interface{}, cap(events))
	go win.EventLoop(events2)
	go uiutil.MouseMoveFilterLoop(events2, events)

	ui := &UI{
		events:  events,
		OnError: func(error) {},
		win:     win,
	}
	ui.Layout.Init(ui)

	return ui, nil
}
func (ui *UI) Close() {
	ui.win.Close()
}

func (ui *UI) HandleEvent(ev interface{}) {
	switch t := ev.(type) {
	case *event.WindowExpose:
		ui.UpdateImageSize()
		ui.Layout.MarkNeedsPaint()
	case *event.WindowInput:
		uiutil.AIE.Apply(ui, &ui.Layout, t.Event, t.Point)
		if ui.AfterInputEvent != nil {
			ui.AfterInputEvent(t.Event, t.Point)
		}
	case *event.WindowPutImageDone:
		ui.onWindowPutImageDone()
	case *UIRunFuncEvent:
		ui.onRunFunc(t)
	case struct{}:
		// no op
	default:
		log.Printf("unhandled event: %#v", ev)
	}
}

func (ui *UI) UpdateImageSize() {
	err := ui.win.UpdateImageSize()
	if err != nil {
		log.Println(err)
	} else {
		ib := ui.win.Image().Bounds()
		if !ui.Layout.Bounds.Eq(ib) {
			ui.Layout.Bounds = ib
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
		if len(ui.events) == 0 {
			// Didn't paint to avoid high fps. Need to ensure a new paint call will happen later by sending a no op event just to allow the loop to iterate.
			ui.EnqueueNoOpEvent()
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
func (ui *UI) onWindowPutImageDone() {
	ui.incompleteDraws--
}

func (ui *UI) EnqueueNoOpEvent() {
	ui.events <- struct{}{}
}
func (ui *UI) RequestPaint() {
	ui.EnqueueNoOpEvent()
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
	if ui.curCursor == c {
		return
	}
	ui.curCursor = c
	ui.win.SetCursor(c)
}

func (ui *UI) QueryPointer() (*image.Point, bool) {
	p, err := ui.win.QueryPointer()
	if err != nil {
		return nil, false
	}
	return p, true
}
func (ui *UI) WarpPointer(p *image.Point) {
	ui.win.WarpPointer(p)
	uiutil.AIE.SetWarpedPointUntilMouseMove(*p)
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

func (ui *UI) GetCPPaste(i event.CopyPasteIndex) (string, error) {
	return ui.win.GetCPPaste(i)
}
func (ui *UI) SetCPCopy(i event.CopyPasteIndex, s string) error {
	return ui.win.SetCPCopy(i, s)
}

func (ui *UI) EnqueueRunFunc(f func()) {
	ui.events <- &UIRunFuncEvent{f}
}
func (ui *UI) onRunFunc(ev *UIRunFuncEvent) {
	ev.F()
}

type UIRunFuncEvent struct {
	F func()
}
