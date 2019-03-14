package drawutil

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/jmigpin/editor/util/iout/iorw"
	"golang.org/x/image/font"
)

type Drawer interface {
	Reader() iorw.Reader
	SetReader(iorw.Reader)
	ContentChanged()

	Face() font.Face
	SetFace(font.Face)
	LineHeight() int
	SetFg(color.Color)

	Bounds() image.Rectangle
	SetBounds(image.Rectangle)

	// rune offset  (set text view position; save/restore view in session file)
	RuneOffset() int
	SetRuneOffset(int)

	LocalPointOf(index int) image.Point
	LocalIndexOf(image.Point) int

	Measure() image.Point
	Draw(img draw.Image)

	// specialized: covers editor row button margin
	FirstLineOffsetX() int
	SetFirstLineOffsetX(x int)

	// cursor
	SetCursorOffset(int)

	// scrollable utils
	ScrollOffset() image.Point
	SetScrollOffset(image.Point)
	ScrollSize() image.Point
	ScrollViewSize() image.Point
	ScrollPageSizeY(up bool) int
	ScrollWheelSizeY(up bool) int

	// visibility utils
	RangeVisible(offset, n int) bool
	RangeVisibleOffset(offset, n int) int
}
