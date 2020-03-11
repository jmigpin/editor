package drawutil

import (
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

//----------

func Baseline(m *font.Metrics) fixed.Int26_6 {
	return m.Ascent
}
func LineHeight(m *font.Metrics) fixed.Int26_6 {
	lh := m.Ascent + m.Descent
	// align with an int to have predictable line positions
	return fixed.I(lh.Ceil())
}
func LineHeightInt(m *font.Metrics) int {
	return LineHeight(m).Floor() // already ceiled at linheight, use floor
}

//----------

func Rect266MinFloorMaxCeil(r fixed.Rectangle26_6) image.Rectangle {
	min := image.Point{r.Min.X.Floor(), r.Min.Y.Floor()}
	max := image.Point{r.Max.X.Ceil(), r.Max.Y.Ceil()}
	return image.Rectangle{min, max}
}

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
