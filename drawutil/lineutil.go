package drawutil

import (
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// space between lines
var margin fixed.Int26_6 = fixed.I(1)

func LineBaseline(fm *font.Metrics) fixed.Int26_6 {
	//return fm.Ascent
	return fm.Height
}
func LineHeight(fm *font.Metrics) fixed.Int26_6 {
	return LineBaseline(fm) + fm.Descent + margin
}
func LineY0(penY fixed.Int26_6, fm *font.Metrics) fixed.Int26_6 {
	return penY - LineBaseline(fm)
}
func LineY1(penY fixed.Int26_6, fm *font.Metrics) fixed.Int26_6 {
	return penY + fm.Descent + margin
}
