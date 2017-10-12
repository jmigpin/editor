package tautil

import (
	"image"

	"golang.org/x/image/math/fixed"
)

type Texta interface {
	Str() string

	EditOpen()
	EditInsert(index int, str string)
	EditDelete(index, index2 int)
	EditClose()

	CursorIndex() int
	SetCursorIndex(int)

	SelectionOn() bool
	SetSelectionOff()
	SelectionIndex() int
	SetSelection(int, int) // selection/cursor indexes

	MakeIndexVisibleAtCenter(int)
	WarpPointerToIndexIfVisible(int)

	OffsetY() fixed.Int26_6
	SetOffsetY(v fixed.Int26_6)
	StrHeight() fixed.Int26_6
	Bounds() image.Rectangle
	LineHeight() fixed.Int26_6
	IndexPoint(int) *fixed.Point26_6
	PointIndex(*fixed.Point26_6) int

	RequestPrimaryPaste() (string, error)
	RequestClipboardPaste() (string, error)
	SetClipboardCopy(string)
	SetPrimaryCopy(string)

	InsertStringAsync(string)
}
