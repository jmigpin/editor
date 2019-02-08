package drawer3

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/jmigpin/editor/util/iout"
	"golang.org/x/image/font"
)

type Drawer interface {
	Reader() iout.Reader
	SetReader(iout.Reader)

	Face() font.Face
	SetFace(font.Face)
	LineHeight() int

	Offset() image.Point
	SetOffset(image.Point)

	Bounds() image.Rectangle
	SetBounds(image.Rectangle)
	SetBoundsSize(image.Point)

	NeedMeasure() bool
	SetNeedMeasure(bool)

	FirstLineOffsetX() int
	SetFirstLineOffsetX(x int)

	// document position
	PointOf(index int) image.Point
	IndexOf(image.Point) int

	// bounds position
	BoundsPointOf(index int) image.Point
	BoundsIndexOf(image.Point) int

	Measure() image.Point // full measure

	Draw(img draw.Image, fg color.Color)
}
