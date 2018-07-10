package drawer3

import (
	"image"

	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/iout"
	"golang.org/x/image/font"
)

type DrawerCommon struct {
	offset           image.Point
	reader           iout.Reader
	face             font.Face
	needMeasure      bool
	bounds           image.Rectangle
	firstLineOffsetX int
}

func (d *DrawerCommon) Offset() image.Point {
	return d.offset
}
func (d *DrawerCommon) SetOffset(o image.Point) {
	d.offset = o
}

func (d *DrawerCommon) Reader() iout.Reader {
	return d.reader
}
func (d *DrawerCommon) SetReader(r iout.Reader) {
	if r != d.reader {
		d.reader = r
		d.needMeasure = true
	}
}

func (d *DrawerCommon) Face() font.Face {
	return d.face
}
func (d *DrawerCommon) SetFace(f font.Face) {
	if f != d.face {
		d.face = f
		d.needMeasure = true
	}
}

func (d *DrawerCommon) Bounds() image.Rectangle {
	return d.bounds
}
func (d *DrawerCommon) SetBounds(r image.Rectangle) {
	d.bounds = r
}

func (d *DrawerCommon) NeedMeasure() bool {
	return d.needMeasure
}
func (d *DrawerCommon) SetNeedMeasure(v bool) {
	d.needMeasure = v
}

func (d *DrawerCommon) FirstLineOffsetX() int {
	return d.firstLineOffsetX
}
func (d *DrawerCommon) SetFirstLineOffsetX(x int) {
	if x != d.firstLineOffsetX {
		d.firstLineOffsetX = x
		d.needMeasure = true
	}
}

func (d *DrawerCommon) LineHeight() int {
	if d.face == nil {
		return 0
	}
	metrics := d.face.Metrics()
	return drawutil.LineHeightInt(&metrics)
}
