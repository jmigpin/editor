package tautil

import (
	"image"

	"golang.org/x/image/math/fixed"
)

type Texta interface {
	Error(error)

	Str() string

	EditOpen()
	EditInsert(index int, str string)
	EditDelete(index int, n int)
	EditClose()

	CursorIndex() int
	SetCursorIndex(int)
	MakeIndexVisible(int)

	OffsetY() fixed.Int26_6
	SetOffsetY(v fixed.Int26_6)
	StrHeight() fixed.Int26_6
	Bounds() *image.Rectangle
	LineHeight() fixed.Int26_6
	IndexPoint(int) *fixed.Point26_6
	PointIndex(*fixed.Point26_6) int

	SelectionOn() bool
	SetSelectionOn(bool)
	SelectionIndex() int
	SetSelectionIndex(int)

	RequestTreePaint()

	RequestClipboardString() (string, error)
	SetClipboardString(string)
}
