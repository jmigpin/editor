package xdriver

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	"log"
	"os"
	"runtime"
	"sync"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/shm"
	"github.com/jezek/xgb/xproto"
	"github.com/jmigpin/editor/driver/xdriver/copypaste"
	"github.com/jmigpin/editor/driver/xdriver/dragndrop"
	"github.com/jmigpin/editor/driver/xdriver/wimage"
	"github.com/jmigpin/editor/driver/xdriver/wmprotocols"
	"github.com/jmigpin/editor/driver/xdriver/xcursors"
	"github.com/jmigpin/editor/driver/xdriver/xinput"
	"github.com/jmigpin/editor/driver/xdriver/xutil"
	"github.com/jmigpin/editor/util/uiutil/event"
)

type Window struct {
	Conn   *xgb.Conn
	Window xproto.Window
	Screen *xproto.ScreenInfo
	GCtx   xproto.Gcontext

	Paste   *copypaste.Paste
	Copy    *copypaste.Copy
	Cursors *xcursors.Cursors
	XInput  *xinput.XInput
	Wmp     *wmprotocols.WMP
	Dnd     *dragndrop.Dnd

	WImg wimage.WImage

	close struct {
		sync.RWMutex
		closing bool
		closed  bool
	}
}

func NewWindow(opt *event.WindowOptions) (*Window, error) {
	display := os.Getenv("DISPLAY")

	// help get a display target
	origDisplay := display
	if display == "" {
		switch runtime.GOOS {
		case "windows":
			display = "127.0.0.1:0.0"
		}
	}

	conn, err := xgb.NewConnDisplay(display)
	if err != nil {
		// improve error with hint
		if origDisplay == "" {
			err = fmt.Errorf("%w (Hint: is x11 running?)", err)
		}
		return nil, fmt.Errorf("x11 conn: %w", err)
	}

	// initialize extensions early to avoid concurrent map read/write (XGB issue)
	wimage.Init(conn)

	win := &Window{Conn: conn}

	if err := win.initialize(opt); err != nil {
		_ = win.Close() // best effort to close since it was opened
		return nil, fmt.Errorf("win init: %w", err)
	}

	return win, nil
}

func (win *Window) initialize(opt *event.WindowOptions) error {
	// Disable xgb logger that prints to stderr
	//xgb.Logger = log.New(ioutil.Discard, "", 0)

	si := xproto.Setup(win.Conn)
	win.Screen = si.DefaultScreen(win.Conn)

	window, err := xproto.NewWindowId(win.Conn)
	if err != nil {
		return err
	}
	win.Window = window

	// event mask
	var evMask uint32 = 0 |
		xproto.EventMaskStructureNotify |
		xproto.EventMaskExposure |
		xproto.EventMaskPropertyChange |
		//xproto.EventMaskPointerMotionHint |
		//xproto.EventMaskButtonMotion |
		xproto.EventMaskPointerMotion |
		xproto.EventMaskButtonPress |
		xproto.EventMaskButtonRelease |
		xproto.EventMaskKeyPress |
		xproto.EventMaskKeyRelease |
		0
	// mask/values order is defined by the protocol
	mask := uint32(xproto.CwEventMask)
	values := []uint32{evMask}

	x, y, w, h := 0, 0, 500, 500
	if !opt.Rect.Empty() {
		x, y = opt.Rect.Min.X, opt.Rect.Min.Y
		w, h = opt.Rect.Dx(), opt.Rect.Dy()
	}

	_ = xproto.CreateWindow(
		win.Conn,
		win.Screen.RootDepth,
		win.Window,
		win.Screen.Root,
		int16(x), int16(y), uint16(w), uint16(h),
		0, // border width
		xproto.WindowClassInputOutput,
		win.Screen.RootVisual,
		mask, values)

	if err := xutil.LoadAtoms(win.Conn, &Atoms, false); err != nil {
		return err
	}

	if opt.StartMaximized {
		data := make([]byte, 8)
		xgb.Put32(data[0:], uint32(Atoms.NetWMStateMaximizedHorz))
		xgb.Put32(data[4:], uint32(Atoms.NetWMStateMaximizedVert))
		_ = xproto.ChangeProperty(
			win.Conn,
			xproto.PropModeReplace,
			win.Window,
			Atoms.NetWMState,
			xproto.AtomAtom,
			32,
			2,
			data)
	}

	_ = xproto.MapWindow(win.Conn, window)

	// graphical context
	gCtx, err := xproto.NewGcontextId(win.Conn)
	if err != nil {
		return err
	}
	win.GCtx = gCtx

	gmask := uint32(0)
	gvalues := []uint32{}
	c2 := xproto.CreateGCChecked(win.Conn, win.GCtx, xproto.Drawable(win.Window), gmask, gvalues)
	if err := c2.Check(); err != nil {
		return err
	}

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

	opt2 := &wimage.Options{win.Conn, win.Window, win.Screen, win.GCtx}
	img, err := wimage.NewWImage(opt2)
	if err != nil {
		return err
	}
	win.WImg = img

	wmp, err := wmprotocols.NewWMP(win.Conn, win.Window)
	if err != nil {
		return err
	}
	win.Wmp = wmp

	return nil
}

//----------

func (win *Window) Close() (rerr error) {
	win.close.Lock()
	defer win.close.Unlock()

	// TODO: closing the image may get memory errors from ongoing draws
	// If a request is called outside the UI loop, using the image will give errors
	//rerr = win.WImg.Close()

	if !win.close.closed {
		win.Conn.Close() // conn.WaitForEvent() will return with (nil,nil)
		win.close.closed = true
	}

	return nil
}

func (win *Window) closeReqFromWindow() error {
	win.close.Lock()
	defer win.close.Unlock()
	win.close.closing = true // no more requests allowed, speeds up closing
	return nil
}

func (win *Window) connClosedPossiblyFromServer() {
	win.close.Lock()
	defer win.close.Unlock()
	win.close.closed = true
}

//----------

func (win *Window) NextEvent() (event.Event, bool) {
	win.close.RLock()
	ok := !win.close.closed
	win.close.RUnlock()
	if !ok {
		return nil, false
	}

	for {
		ev := win.nextEvent2()
		// ev can be nil when the event was consumed internally
		if ev == nil {
			continue
		}
		return ev, true
	}
}

func (win *Window) nextEvent2() any {
	ev, xerr := win.Conn.WaitForEvent()
	if ev == nil {
		if xerr != nil {
			return error(xerr)
		}
		// connection closed: ev==nil && xerr==nil
		win.connClosedPossiblyFromServer()
		return &event.WindowClose{}
	}

	switch t := ev.(type) {
	case xproto.ConfigureNotifyEvent: // structure (position,size,...)
		//x, y := int(t.X), int(t.Y) // commented: must use (0,0)
		w, h := int(t.Width), int(t.Height)
		r := image.Rect(0, 0, w, h)
		return &event.WindowResize{Rect: r}
	case xproto.ExposeEvent: // region needs paint
		//x, y := int(t.X), int(t.Y) // commented: must use (0,0)
		w, h := int(t.Width), int(t.Height)
		r := image.Rect(0, 0, w, h)
		return &event.WindowExpose{Rect: r}
	case xproto.MapNotifyEvent: // window mapped (created)
	case xproto.ReparentNotifyEvent: // window rerooted
	case xproto.MappingNotifyEvent: // keyboard mapping
		if err := win.XInput.ReadMapping(); err != nil {
			return err
		}

	case xproto.UnmapNotifyEvent: // window unmapped (hidden)
		return nil

	case xproto.KeyPressEvent:
		return win.XInput.KeyPress(&t)
	case xproto.KeyReleaseEvent:
		return win.XInput.KeyRelease(&t)
	case xproto.ButtonPressEvent:
		return win.XInput.ButtonPress(&t)
	case xproto.ButtonReleaseEvent:
		return win.XInput.ButtonRelease(&t)
	case xproto.MotionNotifyEvent:
		return win.XInput.MotionNotify(&t)

	case xproto.SelectionNotifyEvent:
		win.Paste.OnSelectionNotify(&t)
		win.Dnd.OnSelectionNotify(&t)
	case xproto.SelectionRequestEvent:
		if err := win.Copy.OnSelectionRequest(&t); err != nil {
			return err
		}
	case xproto.SelectionClearEvent:
		win.Copy.OnSelectionClear(&t)

	case xproto.ClientMessageEvent:
		delWin := win.Wmp.OnClientMessageDeleteWindow(&t)
		if delWin {
			// TODO: won't allow applications to ignore a close request
			// speedup close (won't accept more requests)
			win.closeReqFromWindow()

			return &event.WindowClose{}
		}
		if ev2, err, ok := win.Dnd.OnClientMessage(&t); ok {
			if err != nil {
				return err
			} else {
				return ev2
			}
		}

	case xproto.PropertyNotifyEvent:
		win.Paste.OnPropertyNotify(&t)

	case shm.CompletionEvent:
		win.WImg.PutImageCompleted()

	default:
		log.Printf("unhandled event: %#v", ev)
	}
	return nil
}

//----------

func (win *Window) Request(req event.Request) error {
	// requests that need write lock
	switch req.(type) {
	case *event.ReqClose:
		return win.Close()
	}

	win.close.RLock()
	defer win.close.RUnlock()
	if win.close.closing || win.close.closed {
		return errors.New("window closing/closed")
	}

	switch r := req.(type) {
	case *event.ReqWindowSetName:
		return win.setWindowName(r.Name)
	case *event.ReqWindowMaximize:
		return win.maximizeWindow()
	case *event.ReqImage:
		r.ReplyImg = win.image()
		return nil
	case *event.ReqImagePut:
		return win.WImg.PutImage(r.Rect)
	case *event.ReqImageResize:
		return win.resizeImage(r.Rect)
	case *event.ReqCursorSet:
		return win.setCursor(r.Cursor)
	case *event.ReqPointerQuery:
		p, err := win.queryPointer()
		r.ReplyP = p
		return err
	case *event.ReqPointerWarp:
		return win.warpPointer(r.P)
	case *event.ReqClipboardDataGet:
		s, err := win.Paste.Get(r.Index)
		r.ReplyS = s
		return err
	case *event.ReqClipboardDataSet:
		return win.Copy.Set(r.Index, r.Str)
	default:
		return fmt.Errorf("todo: %T", r)
	}
}

//----------

func (win *Window) setWindowName(str string) error {
	c1 := xproto.ChangePropertyChecked(
		win.Conn,
		xproto.PropModeReplace,
		win.Window,       // requestor window
		Atoms.NetWMName,  // property
		Atoms.Utf8String, // target
		8,                // format
		uint32(len(str)),
		[]byte(str))
	return c1.Check()
}

func (win *Window) maximizeWindow() error {
	ev := xproto.ClientMessageEvent{
		Type:   Atoms.NetWMState,
		Format: 32,
		Window: win.Window,
		Data: xproto.ClientMessageDataUnionData32New([]uint32{
			1, // _NET_WM_STATE_ADD
			uint32(Atoms.NetWMStateMaximizedHorz),
			uint32(Atoms.NetWMStateMaximizedVert),
			1, // source indication: 1=application, 2=pager
			0,
		}),
	}
	err := xproto.SendEventChecked(
		win.Conn,
		false,
		win.Screen.Root,
		xproto.EventMaskSubstructureRedirect|
			xproto.EventMaskSubstructureNotify,
		string(ev.Bytes()),
	).Check()
	return err
}

//----------

//func (win *Window) getGeometry() (*xproto.GetGeometryReply, error) {
//	drawable := xproto.Drawable(win.Window)
//	cookie := xproto.GetGeometry(win.Conn, drawable)
//	return cookie.Reply()
//}

//----------

func (win *Window) image() draw.Image {
	return win.WImg.Image()
}

func (win *Window) resizeImage(r image.Rectangle) error {
	ib := win.image().Bounds()
	if !r.Eq(ib) {
		err := win.WImg.Resize(r)
		if err != nil {
			return err
		}
	}
	return nil
}

//----------

func (win *Window) warpPointer(p image.Point) error {
	// warp pointer only if the window has input focus
	cookie := xproto.GetInputFocus(win.Conn)
	reply, err := cookie.Reply()
	if err != nil {
		return err
	}
	if reply.Focus != win.Window {
		return fmt.Errorf("window not focused")
	}
	c2 := xproto.WarpPointerChecked(
		win.Conn,
		xproto.WindowNone,
		win.Window,
		0, 0, 0, 0,
		int16(p.X), int16(p.Y))
	return c2.Check()
}

func (win *Window) queryPointer() (image.Point, error) {
	cookie := xproto.QueryPointer(win.Conn, win.Window)
	r, err := cookie.Reply()
	if err != nil {
		return image.ZP, err
	}
	p := image.Point{int(r.WinX), int(r.WinY)}
	return p, nil
}

//----------

func (win *Window) setCursor(c event.Cursor) (rerr error) {
	switch c {
	case event.DefaultCursor:
		rerr = win.Cursors.SetDefaultCursor()
	case event.HiddenCursor:
		rerr = win.Cursors.SetHiddenCursor()
	case event.NSResizeCursor:
		rerr = win.Cursors.SetCursor(xcursors.SBVDoubleArrow)
	case event.WEResizeCursor:
		rerr = win.Cursors.SetCursor(xcursors.SBHDoubleArrow)
	case event.CloseCursor:
		rerr = win.Cursors.SetCursor(xcursors.XCursor)
	case event.MoveCursor:
		rerr = win.Cursors.SetCursor(xcursors.Fleur)
	case event.HandCursor:
		rerr = win.Cursors.SetCursor(xcursors.Hand2)
	case event.BeamCursor:
		rerr = win.Cursors.SetCursor(xcursors.XTerm)
	case event.WaitCursor:
		rerr = win.Cursors.SetCursor(xcursors.Watch)
	}
	return
}

//----------

var Atoms struct {
	NetWMState              xproto.Atom `loadAtoms:"_NET_WM_STATE"`
	NetWMStateMaximizedHorz xproto.Atom `loadAtoms:"_NET_WM_STATE_MAXIMIZED_HORZ"`
	NetWMStateMaximizedVert xproto.Atom `loadAtoms:"_NET_WM_STATE_MAXIMIZED_VERT"`
	NetWMName               xproto.Atom `loadAtoms:"_NET_WM_NAME"`
	Utf8String              xproto.Atom `loadAtoms:"UTF8_STRING"`
}
