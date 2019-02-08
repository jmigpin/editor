package drawer4

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/mathutil"
)

type Cursor struct {
	d *Drawer
}

func (c *Cursor) Init() {}

func (c *Cursor) Iter() {
	if !c.d.iters.runeR.isRiExtra() && c.d.Opt.Cursor.On {
		c.iter2()
	}
	_ = c.d.iterNext()
}

func (c *Cursor) iter2() {
	if c.d.st.runeR.ri == c.d.Opt.Cursor.index {
		c.draw()
	}
	// delayed draw
	if c.d.st.cursor.delay != nil {
		c.draw2(c.d.st.cursor.delay.penb, c.d.st.cursor.delay.col)
		c.d.st.cursor.delay = nil
	}
}

func (c *Cursor) End() {}

//----------

func (c *Cursor) draw() {
	// pen bounds
	offset := mathutil.PIntf2(c.d.Offset())
	pos := c.d.Bounds().Min
	penb := c.d.iters.runeR.offsetPenBoundsRect(offset, pos)

	// color
	col := c.d.Opt.Cursor.Fg
	if col == nil {
		col = c.d.st.curColors.fg
	}

	//// draw now
	//c.draw2(penb, col)
	//return

	// delay drawing by one rune to allow drawing the kern bg correctly. The last position is also drawn because the runereader emits a final ru=0 at the end
	c.d.st.cursor.delay = &CursorDelay{penb: penb, col: col}
}

func (c *Cursor) draw2(dr image.Rectangle, col color.Color) {
	img := c.d.st.drawR.img
	bounds := c.d.Bounds()

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
	penb image.Rectangle
	col  color.Color
}
