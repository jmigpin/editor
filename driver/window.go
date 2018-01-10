package driver

import (
	"image"
	"image/draw"

	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type Window interface {
	EventLoop(events chan<- interface{}) // should emit events from uiutil/event

	Close()
	SetWindowName(string)

	Image() draw.Image
	PutImage(*image.Rectangle) error
	UpdateImageSize() error

	SetCursor(widget.Cursor)
	QueryPointer() (*image.Point, error)
	WarpPointer(*image.Point)

	// copypaste
	GetCPPaste(event.CopyPasteIndex) (string, error)
	SetCPCopy(event.CopyPasteIndex, string) error
}
