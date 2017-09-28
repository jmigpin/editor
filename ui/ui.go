package ui

import (
	"image"
	"image/color"
	"image/draw"
	"log"

	"golang.org/x/image/font"

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
	ScrollbarLeft  = false
)

type UI struct {
	win       *Window
	Layout    *Layout
	fface1    font.Face
	CursorMan *CursorMan

	EvReg  *xgbutil.EventRegister
	Events chan interface{}

	incompleteDraws int
}

func NewUI(fface font.Face) (*UI, error) {
	ui := &UI{
		fface1: fface,
		Events: make(chan interface{}, 32),
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
	ui.EvReg.Add(xgbutil.ShmCompletionEventId,
		&xgbutil.ERCallback{ui.onShmCompletion})
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

func (ui *UI) PaintIfNeeded() (painted bool) {
	if ui.incompleteDraws == 0 {
		ui.Layout.C.PaintIfNeeded(func(r *image.Rectangle) {
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

func (ui *UI) RequestPrimaryPaste(requestor interface{}) {
	ui.win.Paste.RequestPrimary(requestor)
}
func (ui *UI) RequestClipboardPaste(requestor interface{}) {
	ui.win.Paste.RequestClipboard(requestor)
}
func (ui *UI) SetClipboardCopy(v string) {
	ui.win.Copy.SetClipboard(v)
}
func (ui *UI) SetPrimaryCopy(v string) {
	ui.win.Copy.SetPrimary(v)
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
	// max size
	maxSize := 5 * 1024 * 1024
	str2 := ta.Str() + str
	if len(str2) > maxSize {
		d := len(str2) - maxSize
		str2 = str2[d:]
	}

	// false,true = keep pos, but clear undo for massive savings
	ta.SetStrClear(str2, false, true)
}

const (
	UITextAreaAppendEventId = xgbutil.UIEventIdStart + iota
)

type UITextAreaAppendEvent struct {
	TextArea *TextArea
	Str      string
}

func SetScrollbarWidth(v int) {
	ScrollbarWidth = v
	SquareWidth = v
}
