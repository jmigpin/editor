package uiutil

import (
	"image"
	"image/draw"
	"log"
	"time"

	"github.com/jmigpin/editor/driver"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
	"golang.org/x/image/font"
)

type BasicUI struct {
	DrawFrameRate int // frames per second
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
		n := ui.RootNode
		if !n.Embed().Bounds.Eq(ib) {
			n.Embed().Bounds = ib
			n.CalcChildsBounds()
			n.Embed().MarkNeedsPaint()
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
		// Didn't paint to avoid high fps.

		if len(ui.events) == 0 {
			// There are no events in the queue that will allow later to check if it is time to paint. Ensure there is one by sending a no-op event to have the loop iterate and call PaintIfTime.
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

	// TODO: review rectangle union performance vs paint rectangles.

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
	err := ui.Win.PutImage(r)
	if err != nil {
		ui.events <- err
	} else {
		ui.incompleteDraws++
	}
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

func (ui *BasicUI) RunOnUIThread(f func()) {
	ui.events <- &UIRunFuncEvent{f}
}

type UIRunFuncEvent struct {
	Func func()
}
