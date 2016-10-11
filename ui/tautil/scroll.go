package tautil

import (
	"golang.org/x/image/math/fixed"
)

func ScrollUp(ta Texta) {
	scroll(ta, -1)
}
func ScrollDown(ta Texta) {
	scroll(ta, 1)
}
func scroll(ta Texta, mult int) {
	lineHeight := ta.LineHeight()
	scrollLines := 4
	v := fixed.Int26_6(scrollLines*mult) * lineHeight
	ta.SetOffsetY(ta.OffsetY() + v)
}
