package xutil

import (
	"errors"
	"fmt"
	"image"
	"io/ioutil"
	"log"

	"github.com/jmigpin/editor/xutil/copypaste"
	"github.com/jmigpin/editor/xutil/dragndrop"
	"github.com/jmigpin/editor/xutil/keybmap"
	"github.com/jmigpin/editor/xutil/xgbutil"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

type XUtil struct {
	Conn     *xgb.Conn
	Window   xproto.Window
	Drawable xproto.Drawable // TODO: remove? use window cast

	SetupInfo *xproto.SetupInfo
	Screen    *xproto.ScreenInfo

	Dnd     *dragndrop.Dnd
	Paste   *copypaste.Paste
	Copy    *copypaste.Copy
	Cursors *Cursors
	KeybMap *keybmap.KeybMap
	ShmWrap *xgbutil.ShmWrap
}

type Event interface{}
type ConnClosedEvent struct{}

var Atoms struct {
	NET_WM_NAME xproto.Atom `loadAtoms:"_NET_WM_NAME"`
	UTF8_STRING xproto.Atom
}

func NewXUtil() (*XUtil, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, err
	}

	// disable xgb logger that prints to stderr
	xgb.Logger = log.New(ioutil.Discard, "", 0)

	xu := &XUtil{Conn: conn}
	if err := xu.init(); err != nil {
		xu.Close()
		return nil, err
	}
	return xu, nil
}
func (xu *XUtil) Close() {
	//_ = xu.Shm.Close()
	xu.Conn.Close()
}
func (xu *XUtil) init() error {
	xu.SetupInfo = xproto.Setup(xu.Conn)
	xu.Screen = xu.SetupInfo.DefaultScreen(xu.Conn)

	if err := xgbutil.LoadAtoms(xu.Conn, &Atoms); err != nil {
		return err
	}

	win, err := xu.createAndMapWindow()
	if err != nil {
		return err
	}
	xu.Window = win
	xu.Drawable = xproto.Drawable(xu.Window)

	xu.setWindowName("Editor")

	k, err := keybmap.NewKeybMap(xu.Conn, xu.SetupInfo)
	if err != nil {
		return err
	}
	xu.KeybMap = k

	dnd, err := dragndrop.NewDnd(xu.Conn, xu.Window)
	if err != nil {
		return err
	}
	xu.Dnd = dnd

	paste, err := copypaste.NewPaste(xu.Conn, xu.Window)
	if err != nil {
		return err
	}
	xu.Paste = paste

	copy, err := copypaste.NewCopy(xu.Conn, xu.Window)
	if err != nil {
		return err
	}
	xu.Copy = copy

	c, err := NewCursors(xu.Conn, xu.Window)
	if err != nil {
		return err
	}
	xu.Cursors = c
	//xu.Cursors.SetCursor(ArrowCursor)

	shmWrap, err := xgbutil.NewShmWrap(xu.Conn, xu.Drawable, xu.Screen.RootDepth)
	if err != nil {
		return err
	}
	xu.ShmWrap = shmWrap

	return nil
}
func (xu *XUtil) createAndMapWindow() (xproto.Window, error) {
	window, err := xproto.NewWindowId(xu.Conn)
	if err != nil {
		return 0, err
	}
	// mask/values order is defined by the protocol
	mask := uint32(
		//xproto.CwBackPixel|
		xproto.CwEventMask,
	)
	values := []uint32{
		//0xffffffff, // white pixel
		//xproto.EventMaskStructureNotify |
		xproto.EventMaskExposure |
			xproto.EventMaskKeyPress |
			xproto.EventMaskButtonPress |
			xproto.EventMaskButtonRelease |
			xproto.EventMaskButtonMotion |
			xproto.EventMaskPointerMotionHint,
	}

	_ = xproto.CreateWindow(
		xu.Conn,
		xu.Screen.RootDepth,
		window,
		xu.Screen.Root,
		0, 0, 500, 500,
		0, // border width
		xproto.WindowClassInputOutput,
		xu.Screen.RootVisual,
		mask, values)

	_ = xproto.MapWindow(xu.Conn, window)

	return window, nil
}
func (xu *XUtil) setWindowName(str string) {
	b := []byte(str)
	_ = xproto.ChangeProperty(
		xu.Conn,
		xproto.PropModeReplace,
		xu.Window,         // requestor window
		Atoms.NET_WM_NAME, // property
		Atoms.UTF8_STRING, // target
		8,                 // format
		uint32(len(b)),
		b)
}
func (xu *XUtil) GetWindowGeometry() (*xproto.GetGeometryReply, error) {
	cookie := xproto.GetGeometry(xu.Conn, xu.Drawable)
	return cookie.Reply()
}
func (xu *XUtil) NewGContext() *GContext {
	return &GContext{xu: xu}
}
func (xu *XUtil) WarpPointer(p *image.Point) {
	// warp pointer only if the window has input focus
	cookie := xproto.GetInputFocus(xu.Conn)
	reply, err := cookie.Reply()
	if err != nil {
		return
	}
	if reply.Focus != xu.Window {
		return
	}
	// warp pointer
	_ = xproto.WarpPointer(
		xu.Conn,
		xproto.WindowNone,
		xu.Window,
		0, 0, 0, 0,
		int16(p.X), int16(p.Y))
}
func (xu *XUtil) RequestMotionNotify() {
	_ = xproto.QueryPointerUnchecked(xu.Conn, xu.Window)
}
func (xu *XUtil) QueryPointer() (*image.Point, bool) {
	cookie := xproto.QueryPointer(xu.Conn, xu.Window)
	r, err := cookie.Reply()
	if err != nil {
		return nil, false
	}
	x := int(r.WinX)
	y := int(r.WinY)
	return &image.Point{x, y}, true
}

func (xu *XUtil) WaitForEvent() (Event, bool) {
	for {
		ev, xerr := xu.Conn.WaitForEvent()
		if ev == nil && xerr == nil { // connection closed
			return nil, false
		}
		if xerr != nil {
			return errors.New(xerr.Error()), true
		}
		ev2 := xu.handleInternalEvents(ev)
		if ev2 == nil {
			continue // event handled internally, get next
		}
		return ev2, true
	}
}
func (xu *XUtil) handleInternalEvents(ev xgb.Event) Event {
	switch ev0 := ev.(type) {
	case xproto.ClientMessageEvent:
		// drag and drop
		ev2, ok, err := xu.Dnd.OnClientMessage(&ev0)
		if err != nil {
			return Event(err)
		}
		if ok {
			return ev2
		}
		// debug unhandled client message event
		name, err := xgbutil.GetAtomName(xu.Conn, ev0.Type)
		if err == nil {
			return fmt.Errorf("unhandled xutil client message event: %+v, ev.type=%v", ev0, name)
		}
	case xproto.SelectionNotifyEvent:
		// paste
		ok := xu.Paste.OnSelectionNotify(&ev0)
		if ok {
			return nil
		}
		// drag and drop
		ok = xu.Dnd.OnSelectionNotify(&ev0)
		if ok {
			return nil
		}
		// debug unhandled selection notify
		name, err := xgbutil.GetAtomName(xu.Conn, ev0.Property)
		if err == nil {
			return fmt.Errorf("unhandled xutil client message event: %+v, ev.peroperty=%v", ev0, name)
		}
	case xproto.SelectionRequestEvent:
		// copy
		ok := xu.Copy.OnSelectionRequest(&ev0)
		if ok {
			return nil
		}
	case xproto.SelectionClearEvent:
		// copy
		ok := xu.Copy.OnSelectionClear(&ev0)
		if ok {
			return nil
		}
	}
	return ev
}
