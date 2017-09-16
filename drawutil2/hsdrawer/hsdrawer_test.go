package hsdrawer

import (
	"image"
	"testing"

	"github.com/jmigpin/editor/drawutil2"
	"github.com/jmigpin/editor/drawutil2/loopers"
)

var loremStr = `Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`

func BenchmarkDraw(b *testing.B) {
	face := drawutil2.GetTestFace()

	img := image.NewRGBA(image.Rect(0, 0, 50000, 50))
	bounds := img.Bounds()

	str := ""
	for i := 0; i < 10; i++ {
		str += loremStr
	}

	d := &HSDrawer{Face: face, Str: str}
	d.CursorIndex = 3
	d.HWordIndex = 15
	d.Selection = &loopers.SelectionIndexes{4, 50}
	c0 := DefaultColors()
	d.Colors = &c0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Draw(img, &bounds)
	}
}
