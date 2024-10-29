package drawutil

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/jmigpin/editor/util/fontutil"
	"github.com/jmigpin/editor/util/iout/iorw"
)

type Drawer interface {
	Reader() iorw.ReaderAt
	SetReader(iorw.ReaderAt)
	ContentChanged()

	FontFace() *fontutil.FontFace
	SetFontFace(*fontutil.FontFace)
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
	RangeVisibleOffset(offset, n int, align RangeAlignment) int
}

//----------

type SyntaxComment struct {
	Start string
	End   string // empty for single line comment
}

func (syc *SyntaxComment) IsLine() bool {
	return syc.End == ""
}

//----------

type RangeAlignment int

const (
	RAlignKeep         RangeAlignment = iota
	RAlignKeepOrBottom                // keep if visible, or bottom
	RAlignAuto
	RAlignTop
	RAlignBottom
	RAlignCenter
)
