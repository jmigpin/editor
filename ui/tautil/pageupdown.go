package tautil

import "golang.org/x/image/math/fixed"

func PageUp(ta Texta) {
	oy := ta.OffsetY()
	oy -= pageSize(ta)
	oy = limitOffsetY(ta, oy)
	ta.SetOffsetY(oy)
}
func PageDown(ta Texta) {
	oy := ta.OffsetY()
	oy += pageSize(ta)
	oy = limitOffsetY(ta, oy)
	ta.SetOffsetY(oy)
}
func pageSize(ta Texta) fixed.Int26_6 {
	b := ta.Bounds()
	return fixed.I(b.Dy())
}
func limitOffsetY(ta Texta, v fixed.Int26_6) fixed.Int26_6 {
	if v < 0 {
		v = 0
	} else if v > ta.StrHeight() {
		v = ta.StrHeight()
	}
	return v
}
