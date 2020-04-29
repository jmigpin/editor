package uiutil

import (
	"fmt"
	"image"
	"image/draw"
	"log"
	"sync"
	"time"

	"github.com/jmigpin/editor/driver"
	"github.com/jmigpin/editor/util/syncutil"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/mousefilter"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type BasicUI struct {
	DrawFrameRate int // frames per second
	RootNode      widget.Node
	Win           driver.Window

	curCursor event.Cursor

	closeOnce sync.Once

	eventsQ *syncutil.SyncedQ // linked list queue (unlimited length)
	applyEv *widget.ApplyEvent
	movef   *mousefilter.MoveFilter
	clickf  *mousefilter.ClickFilter
	dragf   *mousefilter.DragFilter

	pendingPaint   bool
	lastPaintStart time.Time
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

	ui.eventsQ = syncutil.NewSyncedQ()
	ui.applyEv = widget.NewApplyEvent(ui)
	ui.initMouseFilters()

	// Embed nodes have their wrapper nodes set when they are appended to another node. The root node is not appended to any other node, therefore it needs to be set here.
	ui.RootNode = root
	root.Embed().SetWrapperForRoot(root)

	go ui.eventLoop()

	return ui, nil
}

func (ui *BasicUI) initMouseFilters() {
	// move filter
	isMouseMoveEv := func(ev interface{}) bool {
		if wi, ok := ev.(*event.WindowInput); ok {
			if _, ok := wi.Event.(*event.MouseMove); ok {
				return true
			}
		}
		return false
	}
	ui.movef = mousefilter.NewMoveFilter(ui.DrawFrameRate, ui.eventsQ.PushBack, isMouseMoveEv)

	// click/drag filters
	emitFn := func(ev interface{}, p image.Point) {
		ui.handleWidgetEv(ev, p)
	}
	ui.clickf = mousefilter.NewClickFilter(emitFn)
	ui.dragf = mousefilter.NewDragFilter(emitFn)
}

//----------

func (ui *BasicUI) Close() {
	ui.closeOnce.Do(func() {
		if err := ui.Win.Close(); err != nil {
			log.Println(err)
		}
	})
}

//----------

func (ui *BasicUI) eventLoop() {
	for {
		//ui.eventsQ.PushBack(ui.Win.NextEvent()) // slow UI

		ev := ui.Win.NextEvent()
		ui.movef.Filter(ev) // sends events to ui.eventsQ.In()
	}
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
	return ui.eventsQ.PopFront()
}

//----------

func (ui *BasicUI) AppendEvent(ev interface{}) {
	ui.eventsQ.PushBack(ev)
}

//----------

func (ui *BasicUI) HandleEvent(ev interface{}) (handled bool) {
	switch t := ev.(type) {
	case *event.WindowResize:
		ui.resizeImage(t.Rect)
	case *event.WindowExpose:
		ui.RootNode.Embed().MarkNeedsPaint()
	case *event.WindowInput:
		ui.handleWindowInput(t)
	case *UIRunFuncEvent:
		t.Func()
	case *UIPaintTime:
		ui.paint()
	case struct{}:
		// no op, allow layout/schedule funcs to run
	default:
		return false
	}
	return true
}

func (ui *BasicUI) handleWindowInput(wi *event.WindowInput) {
	ui.handleWidgetEv(wi.Event, wi.Point)
	ui.clickf.Filter(wi.Event) // emit events; set on initMouseFilters()
	ui.dragf.Filter(wi.Event)  // emit events; set on initMouseFilters()
}
func (ui *BasicUI) handleWidgetEv(ev interface{}, p image.Point) {
	ui.applyEv.Apply(ui.RootNode, ev, p)
}

//----------

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
	// schedule
	go func() {
		d := ui.durationToNextPaint()
		if d > 0 {
			time.Sleep(d)
		}
		ui.AppendEvent(&UIPaintTime{})
	}()
}

func (ui *BasicUI) durationToNextPaint() time.Duration {
	now := time.Now()
	frameDur := time.Second / time.Duration(ui.DrawFrameRate)
	d := now.Sub(ui.lastPaintStart)
	return frameDur - d
}

//----------

func (ui *BasicUI) paint() {
	// DEBUG: print fps
	now := time.Now()
	//d := now.Sub(ui.lastPaintStart)
	//fmt.Printf("paint: fps %v\n", int(time.Second/d))
	ui.lastPaintStart = now

	ui.paintMarked()
}

func (ui *BasicUI) paintMarked() {
	ui.pendingPaint = false
	u := ui.RootNode.PaintMarked()
	r := u.Intersect(ui.Win.Image().Bounds())
	if !r.Empty() {
		ui.putImage(&r)
	}
}

func (ui *BasicUI) putImage(r *image.Rectangle) {
	if err := ui.Win.PutImage(*r); err != nil {
		ui.AppendEvent(err)
	}
}

//----------

func (ui *BasicUI) EnqueueNoOpEvent() {
	ui.AppendEvent(struct{}{})
}

func (ui *BasicUI) Image() draw.Image {
	return ui.Win.Image()
}

func (ui *BasicUI) WarpPointer(p image.Point) {
	ui.Win.WarpPointer(p)
	//AIE.SetWarpedPointUntilMouseMove(*p) // TODO******
}

func (ui *BasicUI) QueryPointer() (image.Point, error) {
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

func (ui *BasicUI) GetClipboardData(i event.ClipboardIndex, fn func(string, error)) {
	i2 := event.CopyPasteIndex(i)
	ui.Win.GetCPPaste(i2, func(s string, err error) {
		if err != nil {
			ui.AppendEvent(fmt.Errorf("getclipboarddata: %w", err))
		}
		fn(s, err)
	})
}
func (ui *BasicUI) SetClipboardData(i event.ClipboardIndex, s string) {
	i2 := event.CopyPasteIndex(i)
	if err := ui.Win.SetCPCopy(i2, s); err != nil {
		ui.AppendEvent(fmt.Errorf("setclipboarddata: %w", err))
	}
}

//----------

func (ui *BasicUI) RunOnUIGoRoutine(f func()) {
	ui.AppendEvent(&UIRunFuncEvent{f})
}

// Use with care to avoid UI deadlock (waiting within another wait).
func (ui *BasicUI) WaitRunOnUIGoRoutine(f func()) {
	ch := make(chan struct{}, 1)
	ui.RunOnUIGoRoutine(func() {
		f()
		ch <- struct{}{}
	})
	<-ch
}

// Allows triggering a run of applyevent (ex: useful for cursor update, needs point or it won't work).
func (ui *BasicUI) QueueEmptyWindowInputEvent() {
	p, err := ui.QueryPointer()
	if err != nil {
		return
	}
	ui.AppendEvent(&event.WindowInput{Point: p})
}

//----------

type UIPaintTime struct{}

type UIRunFuncEvent struct {
	Func func()
}
