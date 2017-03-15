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

	OffsetY() fixed.Int26_6
	SetOffsetY(v fixed.Int26_6)
	MakeIndexVisible(int)

	SelectionOn() bool
	SetSelectionOn(bool)
	SelectionIndex() int
	SetSelectionIndex(int)

	RequestTreePaint()

	RequestClipboardString() (string, error)
	SetClipboardString(string)

	// used in: movecursor up/down, text area scroll
	LineHeight() fixed.Int26_6
	IndexPoint266(int) *fixed.Point26_6
	Point266Index(*fixed.Point26_6) int
	PointIndexFromOffset(*image.Point) int
}
