package xutil

import (
	"image"
	"log"

	"github.com/jmigpin/editor/xutil/copypaste"
	"github.com/jmigpin/editor/xutil/dragndrop"
	"github.com/jmigpin/editor/xutil/keybmap"
	"github.com/jmigpin/editor/xutil/wmprotocols"
	"github.com/jmigpin/editor/xutil/xgbutil"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

type Window struct {
	Conn       *xgb.Conn
	Window     xproto.Window
	Screen     *xproto.ScreenInfo
	GCtx       xproto.Gcontext
	EvReg      *xgbutil.EventRegister
	Dnd        *dragndrop.Dnd
	Paste      *copypaste.Paste
	Copy       *copypaste.Copy
	Cursors    *Cursors
	KeybMap    *keybmap.KeybMap
	ShmWrap    *xgbutil.ShmWrap
	EventLoopQ chan *xgbutil.ELQEvent
}

func NewWindow() (*Window, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, err
	}
	win := &Window{Conn: conn}
	if err := win.init(); err != nil {
		return nil, err
	}
	return win, nil
}
func (win *Window) init() error {
	// disable xgb logger that prints to stderr
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
			xproto.EventMaskPointerMotion,
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

	k, err := keybmap.NewKeybMap(win.Conn)
	if err != nil {
		return err
	}
	win.KeybMap = k

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

	c, err := NewCursors(win.Conn, win.Window)
	if err != nil {
		return err
	}
	win.Cursors = c
	//_ = win.Cursors.SetCursor(XCXTerm)

	shmWrap, err := xgbutil.NewShmWrap(win.Conn, drawable, win.Screen.RootDepth)
	if err != nil {
		return err
	}
	win.ShmWrap = shmWrap

	win.SetWindowName("Editor")

	win.EvReg = xgbutil.NewEventRegister()

	// setup extensions to use event register
	win.Dnd.SetupEventRegister(win.EvReg)
	win.KeybMap.SetupEventRegister(win.EvReg)
	win.Paste.SetupEventRegister(win.EvReg)
	win.Copy.SetupEventRegister(win.EvReg)

	_, err = wmprotocols.NewWMP(win.Conn, win.Window, win.EvReg)
	if err != nil {
		return err
	}

	//win.EvReg.Add(xproto.MotionNotify,
	//&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
	//_ = xproto.QueryPointerUnchecked(win.Conn, win.Window)
	//}})

	// Test/debug
	n := 0
	win.EvReg.Add(keybmap.KeyPressEventId,
		&xgbutil.ERCallback{func(ev0 xgbutil.EREvent) {
			ev := ev0.(*keybmap.KeyPressEvent)
			if ev.Key.Mods.IsControl() {
				fks := ev.Key.FirstKeysym()
				if fks == 'u' {
					n += 2
					win.Cursors.SetCursor(Cursor(n))
				}
				if fks == 'y' {
					win.Cursors.SetCursor(XCNone)
				}
			}
		}})

	return nil
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
func (win *Window) WarpPointer(p *image.Point) {
	// warp pointer only if the window has input focus
	cookie := xproto.GetInputFocus(win.Conn)
	reply, err := cookie.Reply()
	if err != nil {
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
func (win *Window) QueryPointer() (*image.Point, bool) {
	cookie := xproto.QueryPointer(win.Conn, win.Window)
	r, err := cookie.Reply()
	if err != nil {
		return nil, false
	}
	p := &image.Point{int(r.WinX), int(r.WinY)}
	return p, true
}
func (win *Window) EventLoop() {
	win.EventLoopQ = make(chan *xgbutil.ELQEvent)
	xgbutil.EventLoop(win.Conn, win.EvReg, win.EventLoopQ)

	// cleanup after event loop exit
	err := win.ShmWrap.Close()
	if err != nil {
		log.Println(err)
	}
	win.Conn.Close()
}
func (win *Window) Close() {
	close(win.EventLoopQ)
}

var Atoms struct {
	NET_WM_NAME xproto.Atom `loadAtoms:"_NET_WM_NAME"`
	UTF8_STRING xproto.Atom
}
