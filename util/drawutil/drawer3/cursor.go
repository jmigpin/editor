package drawer3

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/mathutil"
)

type Cursor struct {
	EExt
	Opt CursorOpt
	cc  *CurColors

	// setup values
	img draw.Image

	// start values
	delay *CursorDelay
}

func Cursor1(cc *CurColors) Cursor {
	return Cursor{cc: cc}
}

func (c *Cursor) setup(img draw.Image) {
	c.img = img
}

func (c *Cursor) Start(r *ExtRunner) {
	c.delay = nil
}

func (c *Cursor) Iterate(r *ExtRunner) {
	if r.RR.RiClone() {
		r.NextExt()
		return
	}

	// delay drawing by one rune to allow drawing the kern bg correctly
	// the last position is also drawn because the  runereader emits a final ru=0 at the end
	if c.delay != nil {
		c.draw(c.delay, r)
		c.delay = nil
	}
	if r.RR.Ri == c.Opt.Index { // also runs when ri==eos
		offset := mathutil.PIntf2(r.D.Offset())
		pos := r.D.Bounds().Min
		pb := r.RR.OffsetPenBoundsRect(offset, pos)
		c.delay = &CursorDelay{pb: pb}
	}

	r.NextExt()
}

func (c *Cursor) End(r *ExtRunner) {
	// draw at eos
	if c.delay != nil {
		c.draw(c.delay, r)
	}
}

func (c *Cursor) draw(delay *CursorDelay, r *ExtRunner) {
	// color
	col := c.Opt.Fg
	if col == nil {
		col = c.cc.Fg
	}

	img := c.img
	bounds := r.D.Bounds()
	dr := delay.pb

	// upper square
	r1 := dr
	r1.Min.X -= 1
	r1.Max.X = r1.Min.X + 3
	r1.Max.Y = r1.Min.Y + 3
	r1 = r1.Intersect(bounds)
	imageutil.FillRectangle(img, &r1, col)

	// lower square
	r2 := dr
	r2.Min.X -= 1
	r2.Max.X = r2.Min.X + 3
	r2.Min.Y = r2.Max.Y - 3
	r2 = r2.Intersect(bounds)
	imageutil.FillRectangle(img, &r2, col)

	// vertical bar
	r3 := dr
	r3.Max.X = r3.Min.X + 1
	r3 = r3.Intersect(bounds)
	imageutil.FillRectangle(img, &r3, col)
}

//----------

type CursorDelay struct {
	pb image.Rectangle
}

//----------

type CursorOpt struct {
	Index int
	Fg    color.Color
}
