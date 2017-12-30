package xwindow

import (
	"image"
	"image/draw"
	"log"
	"runtime"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/shm"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil/xcursor"
	"github.com/jmigpin/editor/uiutil/event"
	"github.com/jmigpin/editor/uiutil/widget"
	"github.com/jmigpin/editor/xgbutil"
	"github.com/jmigpin/editor/xgbutil/copypaste"
	"github.com/jmigpin/editor/xgbutil/dragndrop"
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

	done chan struct{}

	Paste        *copypaste.Paste
	Copy         *copypaste.Copy
	Cursors      *xcursors.Cursors
	XInput       *xinput.XInput
	WMP          *wmprotocols.WMP
	Dnd          *dragndrop.Dnd
	ShmImageWrap *shmimage.ShmImageWrap
}

func NewWindow() (*Window, error) {
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
		Conn: conn,
		done: make(chan struct{}, 1),
	}
	if err := win.initialize(); err != nil {
		return nil, errors.Wrap(err, "win init")
	}

	return win, nil
}
func (win *Window) initialize() error {
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
			xproto.EventMaskKeyPress |
			xproto.EventMaskKeyRelease,
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

	xi, err := xinput.NewXInput(win.Conn)
	if err != nil {
		return err
	}
	win.XInput = xi

	dnd, err := dragndrop.NewDnd(win.Conn, win.Window)
	if err != nil {
		return err
	}
	win.Dnd = dnd

	paste, err := copypaste.NewPaste(win.Conn, win.Window)
	if err != nil {
		return err
	}
	win.Paste = paste

	copy, err := copypaste.NewCopy(win.Conn, win.Window)
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

	wmp, err := wmprotocols.NewWMP(win.Conn, win.Window)
	if err != nil {
		return err
	}
	win.WMP = wmp

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

func (win *Window) EventLoop(events chan<- interface{}) {
	for {
		ev, xerr := win.Conn.WaitForEvent()
		if ev == nil && xerr == nil {
			events <- &event.WindowClose{}
			goto forEnd
		}
		if xerr != nil {
			events <- error(xerr)
		}
		if ev != nil {
			switch t := ev.(type) {
			case xproto.ExposeEvent:
				events <- &event.WindowExpose{}
			case shm.CompletionEvent:
				events <- &event.WindowPutImageDone{}

			case xproto.MappingNotifyEvent:
				win.XInput.ReadMapTable()

			case xproto.KeyPressEvent:
				events <- win.XInput.KeyPress(&t)
			case xproto.KeyReleaseEvent:
				events <- win.XInput.KeyRelease(&t)
			case xproto.ButtonPressEvent:
				events <- win.XInput.ButtonPress(&t)
			case xproto.ButtonReleaseEvent:
				events <- win.XInput.ButtonRelease(&t)
			case xproto.MotionNotifyEvent:
				events <- win.XInput.MotionNotify(&t)

			case xproto.SelectionNotifyEvent:
				win.Paste.OnSelectionNotify(&t)
				win.Dnd.OnSelectionNotify(&t)
			case xproto.SelectionRequestEvent:
				win.Copy.OnSelectionRequest(&t)
			case xproto.SelectionClearEvent:
				win.Copy.OnSelectionClear(&t)
			case xproto.ClientMessageEvent:
				win.WMP.OnClientMessage(&t, events)
				win.Dnd.OnClientMessage(&t, events)
			default:
				log.Printf("unhandled event: %#v", ev)
			}
		}
	}
forEnd:

	win.done <- struct{}{}
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
