package driver

import (
	"image"
	"image/draw"

	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type Window interface {
	EventLoop(events chan<- interface{}) // should emit events from uiutil/event

	Close() error
	SetWindowName(string)

	Image() draw.Image
	// if not completed, need to wait for event.WaitPutImageDone
	PutImage(image.Rectangle) (completed bool, _ error)
	ResizeImage(image.Rectangle) error

	SetCursor(widget.Cursor)
	QueryPointer() (*image.Point, error)
	WarpPointer(*image.Point)

	// copypaste
	// paste func arg is called from another goroutine
	GetCPPaste(event.CopyPasteIndex, func(string, error))
	SetCPCopy(event.CopyPasteIndex, string) error
}
