package tautil

import (
	"image"
)

type Texta interface {
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

	RequestPrimaryPaste() (string, error)
	RequestClipboardPaste() (string, error)
	SetClipboardCopy(string)
	SetPrimaryCopy(string)

	InsertStringAsync(string)
}
