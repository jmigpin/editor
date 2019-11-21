package driver

import (
	"image"
	"image/draw"

	"github.com/jmigpin/editor/util/uiutil/event"
)

type Window interface {
	NextEvent() interface{} // emits errors and events (util/uiutil/event)
	//AppendEvent(ev interface{})

	Close() error
	SetWindowName(string)

	Image() draw.Image
	PutImage(image.Rectangle) error
	ResizeImage(image.Rectangle) error

	SetCursor(event.Cursor)
	QueryPointer() (*image.Point, error)
	WarpPointer(*image.Point)

	// copypaste
	// paste func arg is called from another goroutine
	GetCPPaste(event.CopyPasteIndex, func(string, error))
	SetCPCopy(event.CopyPasteIndex, string) error
}
