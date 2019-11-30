package uiutil

import (
	"image"
	"image/draw"
	"log"
	"sync"
	"time"

	"github.com/jmigpin/editor/driver"
	"github.com/jmigpin/editor/util/chanutil"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
	"github.com/pkg/errors"
)

type BasicUI struct {
	DrawFrameRate int // frames per second
	RootNode      widget.Node
	Win           driver.Window
	ApplyEv       *widget.ApplyEvent

	curCursor      event.Cursor
	lastPaintStart time.Time
	lastPaintEnd   time.Time
	pendingPaint   bool

	eventsQ   *chanutil.ChanQ // linked list queue (flexible unlimited length)
	closeOnce sync.Once
}

func NewBasicUI(WinName string, root widget.Node) (*BasicUI, error) {
	win, err := driver.NewWindow()
	if err != nil {
		return nil, err
	}
	win.SetWindowName(WinName)

	ui := &BasicUI{
		DrawFrameRate: 37,
		Win:           win,
	}
	ui.eventsQ = chanutil.NewChanQ(16, 16)
	ui.ApplyEv = widget.NewApplyEvent(ui)

	// Embed nodes have their wrapper nodes set when they are appended to another node. The root node is not appended to any other node, therefore it needs to be set here.
	ui.RootNode = root
	root.Embed().SetWrapperForRoot(root)

	go ui.eventLoop()

	return ui, nil
}

func (ui *BasicUI) Close() {
	ui.closeOnce.Do(func() {
		if err := ui.Win.Close(); err != nil {
			log.Println(err)
		}
	})
}

//----------

func (ui *BasicUI) eventLoop() {
	evQIn := ui.eventsQ.In() // will output events to ui.eventsQ.Out()

	// filter mouvemove events (reduces high fps mouse move events)
	evBridge := make(chan interface{}, cap(evQIn))
	go func() {
		for {
			//evQIn <- ui.Win.NextEvent() // slow UI without mouse filter
			evBridge <- ui.Win.NextEvent()
		}
	}()
	go MouseMoveFilterLoop(evBridge, evQIn, &ui.DrawFrameRate)
}

//----------

// How to use NextEvent():
//
//func SampleEventLoop() {
//	defer ui.Close()
//	for {
//		ev := ui.NextEvent()
//		switch t := ev.(type) {
//		case error:
//			fmt.Println(err)
//		case *event.WindowClose:
//			return
//		default:
//			ui.HandleEvent(ev)
//		}
//		ui.LayoutMarkedAndSchedulePaint()
//	}
//}
func (ui *BasicUI) NextEvent() interface{} {
	return <-ui.eventsQ.Out()
}

//----------

func (ui *BasicUI) AppendEvent(ev interface{}) {
	ui.eventsQ.In() <- ev
}

//----------

func (ui *BasicUI) HandleEvent(ev interface{}) (handled bool) {
	switch t := ev.(type) {
	case *event.WindowResize:
		ui.resizeImage(t.Rect)
	case *event.WindowExpose:
		ui.RootNode.Embed().MarkNeedsPaint()
	case *event.WindowInput:
		ui.ApplyEv.Apply(ui.RootNode, t.Event, t.Point)
	case *UIRunFuncEvent:
		t.Func()
	case *UIPaintTime:
		// paint being done in sync until it ends
		ui.paint()
	case struct{}:
		// no op, allow layout/schedule funcs to run
	default:
		return false
	}
	return true
}

func (ui *BasicUI) LayoutMarkedAndSchedulePaint() {
	ui.RootNode.LayoutMarked()
	ui.schedulePaintMarked()
}

//----------

func (ui *BasicUI) resizeImage(r image.Rectangle) {
	err := ui.Win.ResizeImage(r)
	if err != nil {
		log.Println(err)
		return
	}

	ib := ui.Win.Image().Bounds()
	en := ui.RootNode.Embed()
	if !en.Bounds.Eq(ib) {
		en.Bounds = ib
		en.MarkNeedsLayout()
		en.MarkNeedsPaint()
	}
}

//----------

func (ui *BasicUI) schedulePaintMarked() {
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
		ui.AppendEvent(&UIPaintTime{})
	}()
}

func (ui *BasicUI) durationToNextPaint(now time.Time) time.Duration {
	frameDur := time.Second / time.Duration(ui.DrawFrameRate)
	d := now.Sub(ui.lastPaintStart)
	return frameDur - d
}

//----------

func (ui *BasicUI) paint() {
	ui.pendingPaint = false
	ui.paintMarked()
}

func (ui *BasicUI) paintMarked() {
	u := ui.RootNode.PaintMarked()
	r := u.Intersect(ui.Win.Image().Bounds())
	if !r.Empty() {
		ui.putImage(&r)
	}
}

//----------

func (ui *BasicUI) putImage(r *image.Rectangle) {
	ui.lastPaintStart = time.Now()
	if err := ui.Win.PutImage(*r); err != nil {
		ui.AppendEvent(err)
	}

	//// DEBUG: print fps
	//now := time.Now()
	//d := now.Sub(ui.lastPaintEnd)
	//ui.lastPaintEnd = now
	//fmt.Printf("paint: fps %v\n", int(time.Second/d))
}

//----------

func (ui *BasicUI) EnqueueNoOpEvent() {
	ui.AppendEvent(struct{}{})
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
func (ui *BasicUI) SetCursor(c event.Cursor) {
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
			ui.AppendEvent(errors.Wrap(err, "cppaste"))
		}
		fn(s, err == nil)
	})
}
func (ui *BasicUI) SetCPCopy(i event.CopyPasteIndex, s string) {
	if err := ui.Win.SetCPCopy(i, s); err != nil {
		ui.AppendEvent(errors.Wrap(err, "cpcopy"))
	}
}

//----------

func (ui *BasicUI) RunOnUIGoRoutine(f func()) {
	ui.AppendEvent(&UIRunFuncEvent{f})
}

// Allows triggering a run of applyevent (ex: useful for cursor update).
func (ui *BasicUI) QueueEmptyWindowInputEvent() {
	p, err := ui.QueryPointer()
	if err != nil {
		return
	}
	ui.AppendEvent(&event.WindowInput{Point: *p})
}

//----------

type UIPaintTime struct{}

type UIRunFuncEvent struct {
	Func func()
}
