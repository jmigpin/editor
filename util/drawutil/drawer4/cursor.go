package drawer4

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/util/imageutil"
)

type Cursor struct {
	d *Drawer
}

func (c *Cursor) Init() {}

func (c *Cursor) Iter() {
	if c.d.Opt.Cursor.On {
		if c.d.iters.runeR.isNormal() {
			c.iter2()
		}
	}
	if !c.d.iterNext() {
		return
	}
}

func (c *Cursor) iter2() {
	if c.d.st.runeR.ri == c.d.opt.cursor.offset {
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
	penb := c.d.iters.runeR.penBoundsRect()

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

	vbw := 1 // default vertical bar width

	// vertical bar
	r3 := dr
	r3.Min.X -= vbw / 2
	r3.Max.X = r3.Min.X + vbw
	r3 = r3.Intersect(bounds)
	imageutil.FillRectangle(img, &r3, col)

	// squares width
	aw := vbw // added width
	if c.d.Opt.Cursor.AddedWidth > 0 {
		aw = c.d.Opt.Cursor.AddedWidth
	}
	w := vbw + aw*2 // width

	// upper square
	r1 := r3
	r1.Min.X -= aw
	r1.Max.X += aw
	r1.Max.Y = r1.Min.Y + w
	r1 = r1.Intersect(bounds)
	imageutil.FillRectangle(img, &r1, col)
	// lower square
	r2 := r3
	r2.Min.X -= aw
	r2.Max.X += aw
	r2.Min.Y = r2.Max.Y - w
	r2 = r2.Intersect(bounds)
	imageutil.FillRectangle(img, &r2, col)
}

//----------

type CursorDelay struct {
	penb image.Rectangle
	col  color.Color
}
