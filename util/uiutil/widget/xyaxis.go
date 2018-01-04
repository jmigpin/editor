package widget

import "image"

// Allows calculations to be done X oriented, and have it translated to Y axis.
// Usefull for layouts that want to layout elements in a vertical or horizontal
// direction depending on a flag.
type XYAxis struct {
	YAxis bool
}

func (xy *XYAxis) Point(p *image.Point) *image.Point {
	if xy.YAxis {
		return &image.Point{p.Y, p.X}
	}
	return p
}
func (xy *XYAxis) Rectangle(r *image.Rectangle) *image.Rectangle {
	if xy.YAxis {
		u := image.Rect(r.Min.Y, r.Min.X, r.Max.Y, r.Max.X)
		return &u
	}
	return r
}
func (xy *XYAxis) BoolPair(x, y bool) (bool, bool) {
	if xy.YAxis {
		return y, x
	}
	return x, y
}
