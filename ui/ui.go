package ui

import (
	"image"
	"image/color"
	"image/draw"
	"log"

	"golang.org/x/image/font"

	"github.com/jmigpin/editor/imageutil"
	"github.com/jmigpin/editor/ui/tautil"
	"github.com/jmigpin/editor/uiutil/widget"
	"github.com/jmigpin/editor/xgbutil/evreg"
	"github.com/jmigpin/editor/xgbutil/xcursors"
	"github.com/jmigpin/editor/xgbutil/xinput"
	"github.com/jmigpin/editor/xgbutil/xwindow"

	"github.com/BurntSushi/xgb/xproto"
)

const (
	SeparatorWidth = 1
)

var (
	ScrollbarWidth = 10
	SquareWidth    = 10
	ScrollbarLeft  = false
)

func SetScrollbarAndSquareWidth(v int) {
	ScrollbarWidth = v
	SquareWidth = v
}

type UI struct {
	win       *xwindow.Window
	Layout    *Layout
	fface1    font.Face
	CursorMan *CursorMan

	EvReg   *evreg.Register
	Events2 chan interface{}

	incompleteDraws int
}

func NewUI(fface font.Face) (*UI, error) {
	ui := &UI{
		fface1:  fface,
		Events2: make(chan interface{}, 256),
	}

	ui.EvReg = evreg.NewRegister()
	ui.EvReg.Events = ui.Events2

	win, err := xwindow.NewWindow(ui.EvReg)
	if err != nil {
		return nil, err
	}
	win.SetWindowName("Editor")
	ui.win = win

	ui.CursorMan = NewCursorMan(ui)

	ui.Layout = NewLayout(ui)

	ui.EvReg.Add(xproto.Expose, ui.onExpose)
	ui.EvReg.Add(evreg.ShmCompletionEventId, ui.onShmCompletion)

	// TODO: can be improved with a "pointevent" that as a point to test inbounds
	ui.EvReg.Add(xinput.KeyPressEventId, ui.onKeyPress)
	ui.EvReg.Add(xinput.ButtonPressEventId, ui.onButtonPress)
	ui.EvReg.Add(xinput.ButtonReleaseEventId, ui.onButtonRelease)
	ui.EvReg.Add(xinput.MotionNotifyEventId, ui.onMotionNotify)
	ui.EvReg.Add(xinput.DoubleClickEventId, ui.onDoubleClick)
	ui.EvReg.Add(xinput.TripleClickEventId, ui.onTripleClick)

	ui.EvReg.Add(UITextAreaAppendAsyncEventId, ui.onTextAreaAppendAsync)
	ui.EvReg.Add(UITextAreaInsertStringAsyncEventId, ui.onTextAreaInsertStringAsync)
	ui.EvReg.Add(UIRowDoneExecutingAsyncEventId, ui.onRowDoneExecutingAsync)

	return ui, nil
}
func (ui *UI) Close() {
	ui.win.Close()
}

func (ui *UI) onExpose(ev0 interface{}) {
	ui.UpdateImageSize()
	ui.Layout.MarkNeedsPaint()
}

func (ui *UI) UpdateImageSize() {
	err := ui.win.UpdateImageSize()
	if err != nil {
		log.Println(err)
	} else {
		ib := ui.win.Image().Bounds()
		if !ui.Layout.Bounds().Eq(ib) {
			ui.Layout.SetBounds(&ib)
			ui.Layout.CalcChildsBounds()
			ui.Layout.MarkNeedsPaint()
		}
	}
}

func (ui *UI) PaintIfNeeded() (painted bool) {
	if ui.incompleteDraws == 0 {
		widget.PaintIfNeeded(ui.Layout, func(r *image.Rectangle) {
			painted = true
			ui.incompleteDraws++
			ui.win.PutImage(r)
		})
	}
	return painted
}
func (ui *UI) onShmCompletion(_ interface{}) {
	ui.incompleteDraws--
}

func (ui *UI) onKeyPress(ev0 interface{}) {
	ev := ev0.(*xinput.KeyPressEvent)
	widget.ApplyInputEventInBounds(ui.Layout, ev, *ev.Point)
}
func (ui *UI) onButtonPress(ev0 interface{}) {
	ev := ev0.(*xinput.ButtonPressEvent)
	widget.ApplyInputEventInBounds(ui.Layout, ev, *ev.Point)
}
func (ui *UI) onButtonRelease(ev0 interface{}) {
	ev := ev0.(*xinput.ButtonReleaseEvent)
	widget.ApplyInputEventInBounds(ui.Layout, ev, *ev.Point)
}
func (ui *UI) onMotionNotify(ev0 interface{}) {
	ev := ev0.(*xinput.MotionNotifyEvent)
	widget.ApplyInputEventInBounds(ui.Layout, ev, *ev.Point)
}
func (ui *UI) onDoubleClick(ev0 interface{}) {
	ev := ev0.(*xinput.DoubleClickEvent)
	widget.ApplyInputEventInBounds(ui.Layout, ev, *ev.Point)
}
func (ui *UI) onTripleClick(ev0 interface{}) {
	ev := ev0.(*xinput.TripleClickEvent)
	widget.ApplyInputEventInBounds(ui.Layout, ev, *ev.Point)
}

func (ui *UI) RequestPaint() {
	ui.EvReg.Enqueue(evreg.NoOpEventId, nil)
}

func (ui *UI) Image() draw.Image {
	return ui.win.Image()
}
func (ui *UI) FillRectangle(r *image.Rectangle, c color.Color) {
	imageutil.FillRectangle(ui.Image(), r, c)
}
func (ui *UI) BorderRectangle(r *image.Rectangle, c color.Color, size int) {
	imageutil.BorderRectangle(ui.Image(), r, c, size)
}

// Default fontface (used by textarea)
func (ui *UI) FontFace() font.Face {
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

	pad := 5

	set := func(v *int, min, max int) {
		if max-min < pad*2 {
			*v = min + (max-min)/2
		} else {
			if *v < min+pad {
				*v = min + pad
			} else if *v > max-pad {
				*v = max - pad
			}
		}
	}

	r := *r0
	set(&p.X, r.Min.X, r.Max.X)
	set(&p.Y, r.Min.Y, r.Max.Y)

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
	ui.win.Copy.SetClipboard(v)
}
func (ui *UI) SetPrimaryCopy(v string) {
	ui.win.Copy.SetPrimary(v)
}

func (ui *UI) TextAreaAppendAsync(ta *TextArea, str string) {
	ev := &UITextAreaAppendAsyncEvent{ta, str}
	ui.EvReg.Enqueue(UITextAreaAppendAsyncEventId, ev)
}
func (ui *UI) onTextAreaAppendAsync(ev0 interface{}) {
	ev := ev0.(*UITextAreaAppendAsyncEvent)
	ta := ev.TextArea
	str := ev.Str

	// max size for appends
	maxSize := 5 * 1024 * 1024
	str2 := ta.Str() + str
	if len(str2) > maxSize {
		d := len(str2) - maxSize
		str2 = str2[d:]
	}

	// false,true = keep pos, but clear undo for massive savings
	ta.SetStrClear(str2, false, true)
}

func (ui *UI) TextAreaInsertStringAsync(ta *TextArea, str string) {
	ev := &UITextAreaInsertStringAsyncEvent{ta, str}
	ui.EvReg.Enqueue(UITextAreaInsertStringAsyncEventId, ev)
}
func (ui *UI) onTextAreaInsertStringAsync(ev0 interface{}) {
	ev := ev0.(*UITextAreaInsertStringAsyncEvent)
	tautil.InsertString(ev.TextArea, ev.Str)
}

func (ui *UI) onRowDoneExecutingAsync(ev0 interface{}) {
	ev := ev0.(*UIRowDoneExecutingAsyncEvent)
	ev.Row.Square.SetValue(SquareExecuting, false)
}

const (
	UITextAreaAppendAsyncEventId = evreg.UIEventIdStart + iota
	UITextAreaInsertStringAsyncEventId
	UIRowDoneExecutingAsyncEventId
)

type UITextAreaAppendAsyncEvent struct {
	TextArea *TextArea
	Str      string
}
type UITextAreaInsertStringAsyncEvent struct {
	TextArea *TextArea
	Str      string
}
type UIRowDoneExecutingAsyncEvent struct {
	Row *Row
}
