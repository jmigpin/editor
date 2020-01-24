package driver

import (
	"errors"
	"image"
	"image/draw"

	"github.com/jmigpin/editor/util/uiutil/event"
)

type Window2 interface {
	NextEvent() (_ event.Event, ok bool) // !ok = no more events
	Request(event.Request) error
}

//----------

// Deprecated: use Window2
type Window interface {
	NextEvent() interface{} // emits errors and events (util/uiutil/event)

	Close() error
	SetWindowName(string)

	Image() draw.Image
	PutImage(image.Rectangle) error
	ResizeImage(image.Rectangle) error

	SetCursor(event.Cursor)
	QueryPointer() (image.Point, error)
	WarpPointer(image.Point)

	// copypaste
	// paste func arg is called from another goroutine
	GetCPPaste(event.CopyPasteIndex, func(string, error))
	SetCPCopy(event.CopyPasteIndex, string) error
}

// Deprecated: use NewWindow2
func NewWindow() (Window, error) {
	w2, err := NewWindow2()
	if err != nil {
		return nil, err
	}
	return NewW2Window(w2), nil
}

//----------

// Maintain Window interface with Window2 based implementation.
type W2Window struct {
	W2 Window2
}

func NewW2Window(w2 Window2) *W2Window {
	return &W2Window{w2}
}

func (w *W2Window) NextEvent() interface{} {
	ev, ok := w.W2.NextEvent()
	if !ok {
		return errors.New("no more events")
	}
	return ev
}
func (w *W2Window) Close() error {
	req := &event.ReqClose{}
	return w.W2.Request(req)
}
func (w *W2Window) SetWindowName(name string) {
	req := &event.ReqWindowSetName{name}
	if err := w.W2.Request(req); err != nil {
		// TODO
	}
}
func (w *W2Window) Image() draw.Image {
	req := &event.ReqImage{}
	if err := w.W2.Request(req); err != nil {
		// dummy img to avoid errors
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	return req.ReplyImg
}
func (w *W2Window) PutImage(r image.Rectangle) error {
	req := &event.ReqImagePut{r}
	return w.W2.Request(req)
}
func (w *W2Window) ResizeImage(r image.Rectangle) error {
	req := &event.ReqImageResize{r}
	return w.W2.Request(req)
}
func (w *W2Window) SetCursor(c event.Cursor) {
	req := &event.ReqCursorSet{c}
	if err := w.W2.Request(req); err != nil {
		// TODO
	}
}
func (w *W2Window) QueryPointer() (image.Point, error) {
	req := &event.ReqPointerQuery{}
	err := w.W2.Request(req)
	return req.ReplyP, err
}
func (w *W2Window) WarpPointer(p image.Point) {
	req := &event.ReqPointerWarp{p}
	if err := w.W2.Request(req); err != nil {
		// TODO
	}
}
func (w *W2Window) GetCPPaste(i event.CopyPasteIndex, fn func(string, error)) {
	go func() {
		req := &event.ReqClipboardDataGet{Index: event.ClipboardIndex(i)}
		err := w.W2.Request(req)
		fn(req.ReplyS, err)
	}()
}
func (w *W2Window) SetCPCopy(i event.CopyPasteIndex, s string) error {
	req := &event.ReqClipboardDataSet{Index: event.ClipboardIndex(i), Str: s}
	return w.W2.Request(req)
}
