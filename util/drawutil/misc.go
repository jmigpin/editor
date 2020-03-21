package drawutil

import (
	"image"
)

//----------

// Differs from image.Rectangle.Inset in that it accepts x and y args.
func RectInset(r image.Rectangle, xn, yn int) image.Rectangle {
	if r.Dx() < 2*xn {
		r.Min.X = (r.Min.X + r.Max.X) / 2
		r.Max.X = r.Min.X
	} else {
		r.Min.X += xn
		r.Max.X -= xn
	}
	if r.Dy() < 2*yn {
		r.Min.Y = (r.Min.Y + r.Max.Y) / 2
		r.Max.Y = r.Min.Y
	} else {
		r.Min.Y += yn
		r.Max.Y -= yn
	}
	return r
}

//----------

func Clip(r, s image.Rectangle) image.Rectangle {
	u := r.Intersect(s)
	if u.Empty() {
		return image.Rectangle{r.Min, r.Min}
	}
	return u
}
