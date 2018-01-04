package tautil

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/event"
)

type Texta interface {
	Error(error)

	Str() string

	EditOpen()
	EditInsert(index int, str string)
	EditDelete(index, index2 int)
	EditCloseAfterSetCursor()

	CursorIndex() int
	SetCursorIndex(int)

	SelectionOn() bool
	SetSelectionOff()
	SelectionIndex() int
	SetSelection(int, int) // selection/cursor indexes

	MakeIndexVisibleAtCenter(int)

	GetBounds() image.Rectangle

	OffsetY() int
	SetOffsetY(v int)

	StrHeight() int
	LineHeight() int

	GetPoint(int) image.Point
	GetIndex(*image.Point) int

	GetCPPaste(event.CopyPasteIndex) (string, error)
	SetCPCopy(event.CopyPasteIndex, string) error

	InsertStringAsync(string)

	CommentString() string
}
