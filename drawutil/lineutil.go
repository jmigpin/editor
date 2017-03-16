package drawutil

import (
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// space between lines
//var margin fixed.Int26_6 = fixed.I(1)
var margin fixed.Int26_6 = fixed.I(0)

func LineBaseline(fm *font.Metrics) fixed.Int26_6 {
	//return fm.Ascent
	return fm.Height
}
func LineHeight(fm *font.Metrics) fixed.Int26_6 {
	// make it fixed to an int to avoid round errors between lines
	lh := LineBaseline(fm) + fm.Descent + margin
	return fixed.I(lh.Round())
}
func LineY0(penY fixed.Int26_6, fm *font.Metrics) fixed.Int26_6 {
	return penY - LineBaseline(fm)
}
func LineY1(penY fixed.Int26_6, fm *font.Metrics) fixed.Int26_6 {
	return LineY0(penY, fm) + LineHeight(fm)
}
