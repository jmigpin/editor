package uiutil

import (
	"image"
	"image/draw"
	"log"
	"time"

	"github.com/golang/freetype/truetype"
	"github.com/jmigpin/editor/driver"
	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomedium"
)

type BasicUI struct {
	DrawFrameRate int // frame per second
	RootNode      widget.Node
	Win           driver.Window

	events          chan<- interface{}
	lastPaint       time.Time
	incompleteDraws int
	curCursor       widget.Cursor

	fontFace1 font.Face
}

func NewBasicUI(events chan<- interface{}, WinName string) (*BasicUI, error) {
	win, err := driver.NewWindow()
	if err != nil {
		return nil, err
	}
	win.SetWindowName(WinName)

	ui := &BasicUI{
		DrawFrameRate: 37,
		Win:           win,
		events:        events,
	}

	// slow UI without mouse move filter
	//go ui.Win.EventLoop(events)

	// start window event loop with mousemove event filter
	events2 := make(chan interface{}, cap(events))
	go ui.Win.EventLoop(events2)
	go MouseMoveFilterLoop(events2, events, &ui.DrawFrameRate)

	return ui, nil
}

func (ui *BasicUI) Close() {
	ui.Win.Close()
}

func (ui *BasicUI) HandleEvent(ev interface{}) {
	switch t := ev.(type) {
	case *event.WindowExpose:
		ui.UpdateImageSize()
		ui.RootNode.Embed().MarkNeedsPaint()
	case *event.WindowInput:
		AIE.Apply(ui, ui.RootNode, t.Event, t.Point)
	case *event.WindowPutImageDone:
		ui.onWindowPutImageDone()
	case *UIRunFuncEvent:
		t.Func()
	case struct{}:
		// no op
	default:
		log.Printf("unhandled event: %#v", ev)
	}
}

func (ui *BasicUI) UpdateImageSize() {
	err := ui.Win.UpdateImageSize()
	if err != nil {
		log.Println(err)
	} else {
		ib := ui.Win.Image().Bounds()
		en := ui.RootNode.Embed()
		if !en.Bounds.Eq(ib) {
			en.Bounds = ib
			en.CalcChildsBounds()
			en.MarkNeedsPaint()
		}
	}
}

// This function should be called in the event loop after every event.
func (ui *BasicUI) PaintIfTime() {
	now := time.Now()
	d := now.Sub(ui.lastPaint)
	canPaint := d > (time.Second / time.Duration(ui.DrawFrameRate))
	if canPaint {
		painted := ui.paintIfNeeded()
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

func (ui *BasicUI) paintIfNeeded() (painted bool) {
	// Still painting something else, don't paint now. This function should be called again uppon the draw completion event.
	if ui.incompleteDraws != 0 {
		return false
	}

	var u []*image.Rectangle
	widget.PaintIfNeeded(ui.RootNode, func(r *image.Rectangle) {
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

func (ui *BasicUI) putImage(r *image.Rectangle) {
	ui.incompleteDraws++
	ui.Win.PutImage(r)
}
func (ui *BasicUI) onWindowPutImageDone() {
	ui.incompleteDraws--
}

func (ui *BasicUI) EnqueueNoOpEvent() {
	ui.events <- struct{}{}
}
func (ui *BasicUI) RequestPaint() {
	ui.EnqueueNoOpEvent()
}

func (ui *BasicUI) Image() draw.Image {
	return ui.Win.Image()
}

// Implements widget.CursorContext
func (ui *BasicUI) SetCursor(c widget.Cursor) {
	if ui.curCursor == c {
		return
	}
	ui.curCursor = c
	ui.Win.SetCursor(c)
}

func (ui *BasicUI) WarpPointer(p *image.Point) {
	ui.Win.WarpPointer(p)
	AIE.SetWarpedPointUntilMouseMove(*p)
}

func (ui *BasicUI) QueryPointer() (*image.Point, error) {
	return ui.Win.QueryPointer()
}

func (ui *BasicUI) GetCPPaste(i event.CopyPasteIndex) (string, error) {
	return ui.Win.GetCPPaste(i)
}
func (ui *BasicUI) SetCPCopy(i event.CopyPasteIndex, s string) error {
	return ui.Win.SetCPCopy(i, s)
}

// Implements widget.Context
func (ui *BasicUI) FontFace1() font.Face {
	if ui.fontFace1 == nil {
		// default font
		opt := &truetype.Options{DPI: 0, Size: 14, Hinting: font.HintingFull}
		f, _ := truetype.Parse(gomedium.TTF)
		ui.fontFace1 = drawutil.NewFace(f, opt)
	}
	return ui.fontFace1
}

func (ui *BasicUI) RunOnUIThread(f func()) {
	ui.events <- &UIRunFuncEvent{f}
}

type UIRunFuncEvent struct {
	Func func()
}
