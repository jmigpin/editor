package widget

import (
	"image/draw"

	"github.com/jmigpin/editor/util/uiutil/event"
)

type ImageContext interface {
	Image() draw.Image
}

type CursorContext interface {
	SetCursor(event.Cursor)
}

type ClipboardContext interface {
	GetCPPaste(i event.CopyPasteIndex, fn func(string, bool))
	SetCPCopy(i event.CopyPasteIndex, v string)

	RunOnUIGoRoutine(f func())
}
