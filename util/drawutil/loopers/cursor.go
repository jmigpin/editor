package loopers

import (
	"image"

	"github.com/jmigpin/editor/util/imageutil"
)

type Cursor struct {
	EmbedLooper
	strl   *String
	dl     *Draw
	bounds *image.Rectangle
	index  int
}

func MakeCursor(strl *String, dl *Draw, bounds *image.Rectangle, index int) Cursor {
	return Cursor{strl: strl, dl: dl, bounds: bounds, index: index}
}
func (lpr *Cursor) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		if lpr.strl.IsRiClone() {
			return fn()
		}
		if lpr.strl.Ri == lpr.index {
			lpr.drawCursor()
		}
		return fn()
	})
}
func (lpr *Cursor) drawCursor() {
	img := lpr.dl.Image
	bounds := lpr.dl.Bounds

	// use drawer foreground color
	c := lpr.dl.Fg

	// allow to draw outside the bounds used for drawing text
	bounds2 := *lpr.bounds

	pb := lpr.strl.PenBoundsForImage()
	dr := pb.Add(bounds.Min)

	// upper square
	r1 := dr
	r1.Min.X -= 1
	r1.Max.X = r1.Min.X + 3
	r1.Max.Y = r1.Min.Y + 3
	r1 = r1.Intersect(bounds2)
	imageutil.FillRectangle(img, &r1, c)

	// lower square
	r2 := dr
	r2.Min.X -= 1
	r2.Max.X = r2.Min.X + 3
	r2.Min.Y = r2.Max.Y - 3
	r2 = r2.Intersect(bounds2)
	imageutil.FillRectangle(img, &r2, c)

	// vertical bar
	r3 := dr
	r3.Max.X = r3.Min.X + 1
	r3 = r3.Intersect(bounds2)
	imageutil.FillRectangle(img, &r3, c)
}
