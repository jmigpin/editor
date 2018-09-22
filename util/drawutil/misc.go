package drawutil

import (
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

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
