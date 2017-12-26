package driver

import (
	"image"
	"image/draw"

	"github.com/jmigpin/editor/uiutil/event"
	"github.com/jmigpin/editor/uiutil/widget"
	"github.com/jmigpin/editor/xgbutil/evreg"
)

type Window interface {
	Close()
	SetWindowName(string)

	Image() draw.Image
	PutImage(*image.Rectangle)
	UpdateImageSize() error

	SetCursor(widget.Cursor)
	QueryPointer() (*image.Point, error)
	WarpPointer(*image.Point)

	// copypaste
	GetCPPaste(event.CopyPasteIndex) (string, error)
	SetCPCopy(event.CopyPasteIndex, string) error
}

func NewWindow(evReg *evreg.Register) (Window, error) {
	return NewDriverWindow(evReg)
}
