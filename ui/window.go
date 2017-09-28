package ui

import (
	"image"
	"image/draw"
	"log"
	"runtime"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
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

	evReg  *xgbutil.EventRegister
	events chan<- interface{}

	Dnd          *dragndrop.Dnd
	Paste        *copypaste.Paste
	Copy         *copypaste.Copy
	Cursors      *xcursors.Cursors
	XInput       *xinput.XInput
	ShmImageWrap *shmimage.ShmImageWrap
}

func NewWindow(evReg *xgbutil.EventRegister, events chan<- interface{}) (*Window, error) {
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
		Conn:   conn,
		evReg:  evReg,
		events: events,
	}
	if err := win.init(); err != nil {
		return nil, errors.Wrap(err, "win init")
	}

	go win.eventLoop()

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

	paste, err := copypaste.NewPaste(win.Conn, win.Window, win.evReg, win.events)
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

	win.SetWindowName("Editor")

	return nil
}

func (win *Window) eventLoop() {
	for {
		ev, xerr := win.Conn.WaitForEvent()
		if ev == nil && xerr == nil {
			win.events <- xgbutil.ConnectionClosedEventId
			goto forEnd
		}
		if xerr != nil {
			win.events <- xerr
		} else if ev != nil {
			win.events <- ev
		}
	}
forEnd:
}

func (win *Window) Close() {
	err := win.ShmImageWrap.Close()
	if err != nil {
		log.Println(err)
	}
	win.Conn.Close()
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
func (win *Window) QueryPointer() (*image.Point, bool) {
	cookie := xproto.QueryPointer(win.Conn, win.Window)
	r, err := cookie.Reply()
	if err != nil {
		return nil, false
	}
	p := &image.Point{int(r.WinX), int(r.WinY)}
	return p, true
}

var Atoms struct {
	NET_WM_NAME xproto.Atom `loadAtoms:"_NET_WM_NAME"`
	UTF8_STRING xproto.Atom
}
