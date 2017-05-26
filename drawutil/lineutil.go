package drawutil

import (
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// space between lines
var marginTop = fixed.I(0)
var marginBottom = fixed.I(0)

func LineBaseline(fm *font.Metrics) fixed.Int26_6 {
	return fm.Ascent + marginTop
}
func LineHeight(fm *font.Metrics) fixed.Int26_6 {
	// make it fixed to an int to avoid round errors between lines
	u := LineBaseline(fm) + fm.Descent + marginBottom
	return fixed.I(u.Ceil())
}
func LineY0(penY fixed.Int26_6, fm *font.Metrics) fixed.Int26_6 {
	return penY - LineBaseline(fm)
}
func LineY1(penY fixed.Int26_6, fm *font.Metrics) fixed.Int26_6 {
	return LineY0(penY, fm) + LineHeight(fm)
}
