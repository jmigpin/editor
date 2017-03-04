package imageutil

import "image/color"

// Ex. usage: editor.xutil.cursors
func ColorUint16s(c color.Color) (uint16, uint16, uint16, uint16) {
	r, g, b, a := c.RGBA()
	return uint16(r << 8), uint16(g << 8), uint16(b << 8), uint16(a)
}
