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
	ui.EvReg.Add(UITextAreaAppendAsyncEventId, ui.onTextAreaAppendAsync)
	ui.EvReg.Add(UITextAreaInsertStringAsyncEventId, ui.onTextAreaInsertStringAsync)

	return ui, nil
}
func (ui *UI) Close() {
	ui.win.Close()
	close(ui.Events2)
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
	//ui.animatedWarpPointer(p)
}

//func (ui *UI) animatedWarpPointer(p *image.Point) {
//	p0, ok := ui.QueryPointer()
//	if !ok {
//		ui.win.WarpPointer(p)
//		return
//	}

//	//jump := 50
//	//jumpTime := time.Duration(20 * time.Millisecond)

//	//dx := p.X - p0.X
//	//dy := p.Y - p0.Y
//	//dist := math.Sqrt(float64(dx*dx + dy*dy))
//	//jx := int(float64(jump) * float64(dx) / dist)
//	//jy := int(float64(jump) * float64(dy) / dist)
//	//x, y := p0.X, p0.Y
//	//for u := 0.0; u < dist; u += float64(jump) {
//	//	x, y = x+jx, y+jy
//	//	p2 := image.Point{x, y}
//	//	ui.win.WarpPointer(&p2)
//	//	time.Sleep(jumpTime)
//	//}
//	//ui.win.WarpPointer(p)

//	//return

//	fps := 30
//	frameDur := time.Second / time.Duration(fps)
//	dur := time.Duration(400 * time.Millisecond)
//	now := time.Now()
//	end := now.Add(dur)
//	for ; !now.After(end); now = time.Now() {
//		step := dur - end.Sub(now)
//		step2 := float64(step) / float64(dur)

//		dx := p.X - p0.X
//		x := p0.X + int(float64(dx)*step2)

//		dy := p.Y - p0.Y
//		y := p0.Y + int(float64(dy)*step2)

//		p2 := &image.Point{x, y}

//		ui.win.WarpPointer(p2)

//		time.Sleep(frameDur)
//	}

//	// ensure final position at p
//	ui.win.WarpPointer(p)
//}

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

const (
	UITextAreaAppendAsyncEventId = evreg.UIEventIdStart + iota
	UITextAreaInsertStringAsyncEventId
)

type UITextAreaAppendAsyncEvent struct {
	TextArea *TextArea
	Str      string
}
type UITextAreaInsertStringAsyncEvent struct {
	TextArea *TextArea
	Str      string
}
