package uiutil

import (
	"image"
	"image/draw"
	"log"
	"time"

	"github.com/jmigpin/editor/driver"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
	"github.com/pkg/errors"
)

type BasicUI struct {
	DrawFrameRate int // frames per second
	RootNode      widget.Node
	Win           driver.Window
	ApplyEv       *widget.ApplyEvent

	events          chan<- interface{}
	pendingPaint    bool
	lastPaint       time.Time
	incompleteDraws int
	curCursor       widget.Cursor
}

func NewBasicUI(events chan<- interface{}, WinName string, root widget.Node) (*BasicUI, error) {
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

	ui.ApplyEv = widget.NewApplyEvent(ui)

	// Embed nodes have their wrapper nodes set when they are appended to another node. The root node is not appended to any other node, therefore it needs to be set here.
	ui.RootNode = root
	root.Embed().SetWrapperForRoot(root)

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

//----------

func (ui *BasicUI) HandleEvent(ev interface{}) {
	switch t := ev.(type) {
	case *event.WindowExpose, *UIReviewSize:
		ui.UpdateImageSize()
		ui.RootNode.Embed().MarkNeedsPaint()
	case *event.WindowInput:
		ui.ApplyEv.Apply(ui.RootNode, t.Event, t.Point)
	case *event.WindowPutImageDone:
		ui.onWindowPutImageDone()
	case *UIRunFuncEvent:
		t.Func()
	case *UIPaintTime:
		ui.paint()
	case struct{}:
		// no op
	default:
		log.Printf("basicui: unhandled event: %#v", ev)
	}

	ui.RootNode.LayoutMarked()
	ui.schedulePaintMarked()
}

//----------

func (ui *BasicUI) UpdateImageSize() {
	// don't update size if still drawing, enqueue event to try again later
	if ui.incompleteDraws != 0 {
		ui.events <- &UIReviewSize{}
		return
	}

	err := ui.Win.UpdateImageSize()
	if err != nil {
		log.Println(err)
		return
	}
	ib := ui.Win.Image().Bounds()
	en := ui.RootNode.Embed()
	if !en.Bounds.Eq(ib) {
		en.Bounds = ib
		en.MarkNeedsLayout()
	}
}

//----------

func (ui *BasicUI) schedulePaintMarked() {
	// Still painting something else, don't paint now. This function should be called again uppon the draw completion event.
	if ui.incompleteDraws != 0 {
		return
	}

	if ui.RootNode.Embed().TreeNeedsPaint() {
		ui.schedulePaint()
	}
}
func (ui *BasicUI) schedulePaint() {
	if ui.pendingPaint {
		return
	}

	ui.pendingPaint = true
	go func() {
		d := ui.durationToNextPaint(time.Now())
		if d > 0 {
			time.Sleep(d)
		}
		ui.events <- &UIPaintTime{}
	}()
}

func (ui *BasicUI) durationToNextPaint(now time.Time) time.Duration {
	frameDur := time.Second / time.Duration(ui.DrawFrameRate)
	return frameDur - now.Sub(ui.lastPaint)
}

//----------

func (ui *BasicUI) paint() {
	//// DEBUG
	//d := time.Now().Sub(ui.lastPaint)
	//fmt.Printf("paint: fps %v\n", int(time.Second/d))

	ui.pendingPaint = false
	ui.lastPaint = time.Now()
	ui.paintMarked()
}

func (ui *BasicUI) paintMarked() {
	u := ui.RootNode.PaintMarked()
	r := u.Intersect(ui.Win.Image().Bounds())
	if !r.Empty() {
		//log.Printf("putimage %v", r)
		ui.putImage(&r)
	}
}

//----------

func (ui *BasicUI) putImage(r *image.Rectangle) {
	err := ui.Win.PutImage(r)
	if err != nil {
		ui.events <- err
		return
	}
	ui.incompleteDraws++
}

func (ui *BasicUI) onWindowPutImageDone() {
	ui.incompleteDraws--
}

//----------

func (ui *BasicUI) EnqueueNoOpEvent() {
	ui.events <- struct{}{}
}

func (ui *BasicUI) Image() draw.Image {
	return ui.Win.Image()
}

func (ui *BasicUI) WarpPointer(p *image.Point) {
	ui.Win.WarpPointer(p)
	//AIE.SetWarpedPointUntilMouseMove(*p) // TODO******
}

func (ui *BasicUI) QueryPointer() (*image.Point, error) {
	return ui.Win.QueryPointer()
}

//----------

// Implements widget.CursorContext
func (ui *BasicUI) SetCursor(c widget.Cursor) {
	if ui.curCursor == c {
		return
	}
	ui.curCursor = c
	ui.Win.SetCursor(c)
}

//----------

func (ui *BasicUI) GetCPPaste(i event.CopyPasteIndex, fn func(string, bool)) {
	ui.Win.GetCPPaste(i, func(s string, err error) {
		if err != nil {
			ui.events <- errors.Wrap(err, "cppaste")
		}
		fn(s, err == nil)
	})
}
func (ui *BasicUI) SetCPCopy(i event.CopyPasteIndex, s string) {
	if err := ui.Win.SetCPCopy(i, s); err != nil {
		ui.events <- errors.Wrap(err, "cpcopy")
	}
}

//----------

func (ui *BasicUI) RunOnUIGoRoutine(f func()) {
	ui.events <- &UIRunFuncEvent{f}
}

//----------

type UIReviewSize struct{}

type UIPaintTime struct{}

type UIRunFuncEvent struct {
	Func func()
}
