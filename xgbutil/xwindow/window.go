package xwindow

import (
	"image"
	"image/draw"
	"log"
	"runtime"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil/xcursor"
	"github.com/jmigpin/editor/uiutil/event"
	"github.com/jmigpin/editor/uiutil/widget"
	"github.com/jmigpin/editor/xgbutil"
	"github.com/jmigpin/editor/xgbutil/copypaste"
	"github.com/jmigpin/editor/xgbutil/dragndrop"
	"github.com/jmigpin/editor/xgbutil/evreg"
	"github.com/jmigpin/editor/xgbutil/shmimage"
	"github.com/jmigpin/editor/xgbutil/wmprotocols"
	"github.com/jmigpin/editor/xgbutil/xcursors"
	"github.com/jmigpin/editor/xgbutil/xinput"
	"github.com/pkg/errors"
)

type Window struct {
	Conn   *xgb.Conn
	Window xproto.Window
	Screen *xproto.ScreenInfo
	GCtx   xproto.Gcontext

	evReg *evreg.Register
	done  chan struct{}

	Dnd          *dragndrop.Dnd
	Paste        *copypaste.Paste
	Copy         *copypaste.Copy
	Cursors      *xcursors.Cursors
	XInput       *xinput.XInput
	ShmImageWrap *shmimage.ShmImageWrap
}

func NewWindow(evReg *evreg.Register) (*Window, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		if runtime.GOOS == "darwin" {
			msg := err.Error() + ": macOS might need XQuartz installed"
			err = errors.WithMessage(err, msg)
		}
		err2 := errors.Wrap(err, "x conn")
		return nil, err2
	}
	win := &Window{
		Conn:  conn,
		evReg: evReg,
		done:  make(chan struct{}, 1),
	}
	if err := win.init(); err != nil {
		return nil, errors.Wrap(err, "win init")
	}

	go win.eventLoop()

	return win, nil
}
func (win *Window) init() error {
	// Disable xgb logger that prints to stderr
	// Prevents error msg on clean exit when testing in race mode
	// "XGB: xgb.go:526: Invalid event/error type: <nil>"
	// this is an issue with xgb not exiting cleanly on conn.close
	//xgb.Logger = log.New(ioutil.Discard, "", 0)

	si := xproto.Setup(win.Conn)
	win.Screen = si.DefaultScreen(win.Conn)

	window, err := xproto.NewWindowId(win.Conn)
	if err != nil {
		return err
	}
	win.Window = window

	// mask/values order is defined by the protocol
	mask := uint32(
		//xproto.CwBackPixel |
		xproto.CwEventMask,
	)
	values := []uint32{
		//0xffffffff, // white pixel
		//xproto.EventMaskStructureNotify |
		xproto.EventMaskExposure |
			//xproto.EventMaskPointerMotionHint |
			//xproto.EventMaskButtonMotion |
			xproto.EventMaskPointerMotion |
			xproto.EventMaskButtonPress |
			xproto.EventMaskButtonRelease |
			xproto.EventMaskKeyPress,
	}

	_ = xproto.CreateWindow(
		win.Conn,
		win.Screen.RootDepth,
		win.Window,
		win.Screen.Root,
		0, 0, 500, 500,
		0, // border width
		xproto.WindowClassInputOutput,
		win.Screen.RootVisual,
		mask, values)

	_ = xproto.MapWindow(win.Conn, window)

	if err := xgbutil.LoadAtoms(win.Conn, &Atoms); err != nil {
		return err
	}

	// graphical context
	gCtx, err := xproto.NewGcontextId(win.Conn)
	if err != nil {
		return err
	}
	win.GCtx = gCtx
	drawable := xproto.Drawable(win.Window)
	var gmask uint32
	var gvalues []uint32
	_ = xproto.CreateGC(win.Conn, win.GCtx, drawable, gmask, gvalues)

	xi, err := xinput.NewXInput(win.Conn, win.evReg)
	if err != nil {
		return err
	}
	win.XInput = xi

	dnd, err := dragndrop.NewDnd(win.Conn, win.Window, win.evReg)
	if err != nil {
		return err
	}
	win.Dnd = dnd

	paste, err := copypaste.NewPaste(win.Conn, win.Window, win.evReg)
	if err != nil {
		return err
	}
	win.Paste = paste

	copy, err := copypaste.NewCopy(win.Conn, win.Window, win.evReg)
	if err != nil {
		return err
	}
	win.Copy = copy

	c, err := xcursors.NewCursors(win.Conn, win.Window)
	if err != nil {
		return err
	}
	win.Cursors = c

	shmImageWrap, err := shmimage.NewShmImageWrap(win.Conn, drawable, win.Screen.RootDepth)
	if err != nil {
		return err
	}
	win.ShmImageWrap = shmImageWrap

	_, err = wmprotocols.NewWMP(win.Conn, win.Window, win.evReg)
	if err != nil {
		return err
	}

	return nil
}

func (win *Window) Close() {
	err := win.ShmImageWrap.Close()
	if err != nil {
		log.Println(err)
	}
	win.Conn.Close()

	// xgb is not exiting cleanly otherwise this should not block
	// <-win.done
}

func (win *Window) eventLoop() {
	events := make(chan interface{}, 8)

	go func() {
		for {
			ev, xerr := win.Conn.WaitForEvent()
			if ev == nil && xerr == nil {
				events <- &evreg.EventWrap{evreg.ConnectionClosedEventId, nil}
				close(events)
				goto forEnd
			}
			if xerr != nil {
				events <- &evreg.EventWrap{evreg.ErrorEventId, xerr}
			} else if ev != nil {
				eid := evreg.XgbEventId(ev)
				events <- &evreg.EventWrap{eid, ev}
			}
		}
	forEnd:
	}()

	win.motionEventFilterLoop(events, win.evReg.Events)

	win.done <- struct{}{}
}

func (win *Window) motionEventFilterLoop(in <-chan interface{}, out chan<- interface{}) {
	var lastMotionEv interface{}
	var ticker *time.Ticker
	var timeToSend <-chan time.Time

	//n := 0
	keepMotionEv := func(ev interface{}) {
		//n++
		lastMotionEv = ev
		if ticker == nil {
			ticker = time.NewTicker(time.Second / 40)
			timeToSend = ticker.C
		}
	}

	sendMotionEv := func() {
		//log.Printf("kept %d times before sending", n)
		//n = 0
		ticker.Stop()
		ticker = nil
		timeToSend = nil
		out <- lastMotionEv
	}

	sendMotionEvIfKept := func() {
		if ticker != nil {
			sendMotionEv()
		}
	}

	for {
		select {
		case ev, ok := <-in:
			if !ok {
				goto forEnd
			}
			evw := ev.(*evreg.EventWrap)
			switch evw.Event.(type) {
			case xproto.MotionNotifyEvent:
				keepMotionEv(evw)
			default:
				sendMotionEvIfKept()
				out <- evw
			}
		case <-timeToSend:
			sendMotionEv()
		}
	}
forEnd:
}

func (win *Window) SetWindowName(str string) {
	b := []byte(str)
	_ = xproto.ChangeProperty(
		win.Conn,
		xproto.PropModeReplace,
		win.Window,        // requestor window
		Atoms.NET_WM_NAME, // property
		Atoms.UTF8_STRING, // target
		8,                 // format
		uint32(len(b)),
		b)
}
func (win *Window) GetGeometry() (*xproto.GetGeometryReply, error) {
	drawable := xproto.Drawable(win.Window)
	cookie := xproto.GetGeometry(win.Conn, drawable)
	return cookie.Reply()
}

func (win *Window) Image() draw.Image {
	return win.ShmImageWrap.Image()
}
func (win *Window) PutImage(rect *image.Rectangle) {
	win.ShmImageWrap.PutImage(win.GCtx, rect)
}
func (win *Window) UpdateImageSize() error {
	geom, err := win.GetGeometry()
	if err != nil {
		return err
	}
	w, h := int(geom.Width), int(geom.Height)

	r := image.Rect(0, 0, w, h)
	ib := win.Image().Bounds()
	if !r.Eq(ib) {
		err := win.ShmImageWrap.NewImage(&r)
		if err != nil {
			return err
		}
	}
	return nil
}

func (win *Window) WarpPointer(p *image.Point) {
	// warp pointer only if the window has input focus
	cookie := xproto.GetInputFocus(win.Conn)
	reply, err := cookie.Reply()
	if err != nil {
		log.Print(err)
		return
	}
	if reply.Focus != win.Window {
		return
	}
	_ = xproto.WarpPointer(
		win.Conn,
		xproto.WindowNone,
		win.Window,
		0, 0, 0, 0,
		int16(p.X), int16(p.Y))
}
func (win *Window) QueryPointer() (*image.Point, error) {
	cookie := xproto.QueryPointer(win.Conn, win.Window)
	r, err := cookie.Reply()
	if err != nil {
		return nil, err
	}
	p := &image.Point{int(r.WinX), int(r.WinY)}
	return p, nil
}

func (win *Window) GetCPPaste(i event.CopyPasteIndex) (string, error) {
	return win.Paste.Get(i)
}
func (win *Window) SetCPCopy(i event.CopyPasteIndex, v string) error {
	return win.Copy.Set(i, v)
}

func (win *Window) SetCursor(c widget.Cursor) {
	sc := func(c2 xcursors.Cursor) {
		err := win.Cursors.SetCursor(c2)
		if err != nil {
			log.Print(err)
		}
	}
	switch c {
	case widget.NoneCursor:
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

var Atoms struct {
	NET_WM_NAME xproto.Atom `loadAtoms:"_NET_WM_NAME"`
	UTF8_STRING xproto.Atom
}
