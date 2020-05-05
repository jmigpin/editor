package widget

import (
	"image/draw"

	"github.com/jmigpin/editor/v2/util/uiutil/event"
)

type UIContext interface {
	Error(error)

	ImageContext
	CursorContext
	//	Image() draw.Image // TODO
	//	SetCursor(event.Cursor) // TODO

	RunOnUIGoRoutine(f func())
	SetClipboardData(event.ClipboardIndex, string)
	GetClipboardData(event.ClipboardIndex, func(string, error))
}

type ImageContext interface {
	Image() draw.Image
}

type CursorContext interface {
	SetCursor(event.Cursor)
}
