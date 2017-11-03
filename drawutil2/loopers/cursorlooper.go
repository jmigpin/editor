package loopers

import (
	"image"

	"github.com/jmigpin/editor/imageutil"
)

type CursorLooper struct {
	EmbedLooper
	strl        *StringLooper
	dl          *DrawLooper
	bounds      *image.Rectangle
	CursorIndex int
}

func NewCursorLooper(strl *StringLooper, dl *DrawLooper, bounds *image.Rectangle) *CursorLooper {
	return &CursorLooper{strl: strl, dl: dl, bounds: bounds}
}
func (lpr *CursorLooper) Loop(fn func() bool) {
	ci := lpr.CursorIndex
	lpr.OuterLooper().Loop(func() bool {
		if lpr.strl.RiClone {
			return fn()
		}
		if lpr.strl.Ri == ci {
			lpr.drawCursor()
		}
		return fn()
	})
	// draw past last position if at str len
	if !lpr.strl.RiClone && lpr.strl.Ri == ci && ci == len(lpr.strl.Str) {
		lpr.drawCursor()
	}
}
func (lpr *CursorLooper) drawCursor() {
	img := lpr.dl.Image
	bounds := lpr.dl.Bounds

	// use drawer foreground color
	c := lpr.dl.Fg
	if c == nil {
		panic("cursor color is nil")
	}

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
