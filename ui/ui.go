package ui

import (
	"image"
	"image/color"
	"image/draw"
	"log"

	"github.com/jmigpin/editor/drawutil"
	"github.com/jmigpin/editor/imageutil"
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

	EvReg  *xgbutil.EventRegister
	Events chan interface{}
}

func NewUI(fface *drawutil.Face) (*UI, error) {
	ui := &UI{
		fface1: fface,
		Events: make(chan interface{}, 50),
		EvReg:  xgbutil.NewEventRegister(),
	}

	win, err := NewWindow(ui.EvReg, ui.Events)
	if err != nil {
		return nil, err
	}
	ui.win = win

	ui.CursorMan = NewCursorMan(ui)

	ui.Layout = NewLayout(ui)

	ui.EvReg.Add(xproto.Expose,
		&xgbutil.ERCallback{ui.onExpose})

	ui.EvReg.Add(UITextAreaAppendEventId,
		&xgbutil.ERCallback{ui.onTextAreaAppend})

	return ui, nil
}
func (ui *UI) Close() {
	ui.win.Close()
	close(ui.Events)
}

func (ui *UI) onExpose(ev0 interface{}) {
	ui.UpdateImageSize()
	ui.Layout.C.NeedPaint()
}

func (ui *UI) UpdateImageSize() {
	err := ui.win.UpdateImageSize()
	if err != nil {
		log.Println(err)
	} else {
		ib := ui.win.Image().Bounds()
		if !ui.Layout.C.Bounds.Eq(ib) {
			ui.Layout.C.Bounds = ib
			ui.Layout.C.CalcChildsBounds()
			ui.Layout.C.NeedPaint()
		}
	}
}

func (ui *UI) PaintIfNeeded() {
	ui.Layout.C.PaintIfNeeded(func(r *image.Rectangle) {
		ui.win.PutImage(r)
	})
}

//func (ui *UI) Paint() {
//ui.win.PutImage(&ui.Layout.C.Bounds)
//}

func (ui *UI) RequestPaint() {
	go func() {
		ui.Events <- xgbutil.NoOpEventId
	}()
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
	// run concurrently so it can be called from the ui thread as well, otherwise it can block
	go func() {
		ev := &xgbutil.EREventData{
			UITextAreaAppendEventId,
			&UITextAreaAppendEvent{ta, str},
		}
		ui.Events <- ev
	}()
}

func (ui *UI) onTextAreaAppend(ev0 interface{}) {
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
