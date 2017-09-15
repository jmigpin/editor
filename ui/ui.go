package ui

import (
	"image"
	"image/color"
	"image/draw"
	"log"

	"github.com/jmigpin/editor/drawutil"
	"github.com/jmigpin/editor/imageutil"
	"github.com/jmigpin/editor/uiutil"
	"github.com/jmigpin/editor/xgbutil"
	"github.com/jmigpin/editor/xgbutil/xcursors"

	"github.com/BurntSushi/xgb/xproto"
)

const (
	SeparatorWidth = 1
)

var (
	ScrollbarWidth = 10
	SquareWidth    = 10
)

type UI struct {
	win       *Window
	Layout    *Layout
	fface1    *drawutil.Face
	CursorMan *CursorMan
	EvReg     *xgbutil.EventRegister
}

func NewUI(fface *drawutil.Face) (*UI, error) {
	ui := &UI{
		fface1: fface,
	}

	win, err := NewWindow()
	if err != nil {
		return nil, err
	}
	ui.win = win
	ui.EvReg = ui.win.EvReg

	ui.CursorMan = NewCursorMan(ui)

	ui.Layout = NewLayout(ui)

	ui.win.EvReg.Add(xproto.Expose,
		&xgbutil.ERCallback{ui.onExpose})
	ui.win.EvReg.Add(xgbutil.QueueEmptyEventId,
		&xgbutil.ERCallback{ui.onQueueEmpty})

	ui.EvReg.Add(UITextAreaAppendEventId,
		&xgbutil.ERCallback{ui.onTextAreaAppend})

	return ui, nil
}
func (ui *UI) Close() {
	ui.win.Close()
}
func (ui *UI) EventLoop() {
	ui.win.RunEventLoop()
}
func (ui *UI) onExpose(ev0 xgbutil.EREvent) {
	ev := ev0.(xproto.ExposeEvent)

	// number of expose events to come
	if ev.Count > 0 {
		//// repaint just the exposed area
		//x0, y0 := int(ev.X), int(ev.Y)
		//x1, y1 := x0+int(ev.Width), y0+int(ev.Height)
		//r := image.Rect(x0, y0, x1, y1)
		//ui.PutImage(&r)

		return // wait for expose with count 0
	}

	err := ui.win.UpdateImageSize()
	if err != nil {
		log.Println(err)
	} else {
		ib := ui.win.Image().Bounds()
		if !ui.Layout.C.Bounds.Eq(ib) {
			ui.Layout.C.Bounds = ib
			ui.Layout.C.CalcChildsBounds()
		}
	}

	ui.Layout.C.NeedPaint()
}

func (ui *UI) onQueueEmpty(ev xgbutil.EREvent) {
	// paint after all events have been handled
	ui.Layout.C.PaintTreeIfNeeded(func(c *uiutil.Container) {
		// paint only the top container of the needed area
		ui.win.PutImage(&c.Bounds)
	})
}

// Send paint request to the main event loop.
// Usefull for async methods that need to be painted.
func (ui *UI) RequestTreePaint() {
	ui.win.EventLoop.EnqueueQEmptyEventIfConnQEmpty()
}

func (ui *UI) Image() draw.Image {
	return ui.win.Image()
}
func (ui *UI) FillRectangle(r *image.Rectangle, c color.Color) {
	imageutil.FillRectangle(ui.Image(), r, c)
}

// Default fontface (used by textarea)
func (ui *UI) FontFace() *drawutil.Face {
	return ui.fface1
}

func (ui *UI) QueryPointer() (*image.Point, bool) {
	return ui.win.QueryPointer()
}
func (ui *UI) WarpPointer(p *image.Point) {
	ui.win.WarpPointer(p)
}
func (ui *UI) WarpPointerToRectanglePad(r0 *image.Rectangle) {
	p, ok := ui.QueryPointer()
	if !ok {
		return
	}
	// pad rectangle
	pad := 25
	r := *r0
	if r.Dx() < pad*2 {
		r.Min.X = r.Min.X + r.Dx()/2
		r.Max.X = r.Min.X
	} else {
		r.Min.X += pad
		r.Max.X -= pad
	}
	if r.Dy() < pad*2 {
		r.Min.Y = r.Min.Y + r.Dy()/2
		r.Max.Y = r.Min.Y
	} else {
		r.Min.Y += pad
		r.Max.Y -= pad
	}
	// put p inside
	if p.Y < r.Min.Y {
		p.Y = r.Min.Y
	} else if p.Y >= r.Max.Y {
		p.Y = r.Max.Y
	}
	if p.X < r.Min.X {
		p.X = r.Min.X
	} else if p.X >= r.Max.X {
		p.X = r.Max.X
	}
	ui.WarpPointer(p)
}

func (ui *UI) SetCursor(c xcursors.Cursor) {
	ui.win.Cursors.SetCursor(c)
}

func (ui *UI) RequestPrimaryPaste() (string, error) {
	return ui.win.Paste.RequestPrimary()
}
func (ui *UI) RequestClipboardPaste() (string, error) {
	return ui.win.Paste.RequestClipboard()
}
func (ui *UI) SetClipboardCopy(v string) {
	ui.win.Copy.Set(v)
}

func (ui *UI) TextAreaAppendAsync(ta *TextArea, str string) {
	// run in go routine so it can be called from the ui thread as well, otherwise it will block
	go func() {
		ev := &UITextAreaAppendEvent{ta, str}
		ui.win.EventLoop.Enqueue(UITextAreaAppendEventId, ev)
		// TODO: enforcing a nil event at end to have a draw - should not be needed with a node that triggers need paint at root
		ui.win.EventLoop.Enqueue(xgbutil.QueueEmptyEventId, nil)
	}()
}
func (ui *UI) onTextAreaAppend(ev0 xgbutil.EREvent) {
	ev := ev0.(*UITextAreaAppendEvent)
	ui.textAreaAppend(ev.TextArea, ev.Str)
}
func (ui *UI) textAreaAppend(ta *TextArea, str string) {
	// cap max size
	maxSize := 1024 * 1024 * 5
	str = ta.Str() + str
	if len(str) > maxSize {
		d := len(str) - maxSize
		str = str[d:]
	}
	// false,true = keep pos, but clear undo for massive savings
	ta.SetStrClear(str, false, true)
}

const (
	UITextAreaAppendEventId = 1300 + iota
)

type UITextAreaAppendEvent struct {
	TextArea *TextArea
	Str      string
}

func SetScrollbarWidth(v int) {
	ScrollbarWidth = v
	SquareWidth = v
}
