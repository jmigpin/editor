package hsdrawer

import (
	"image"
	"testing"

	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/drawutil/loopers"
)

var loremStr = `Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`

func BenchmarkDraw(b *testing.B) {
	f1 := drawutil.GetTestFace()
	f2 := drawutil.NewFaceRunes(f1)
	f3 := drawutil.NewFaceCache(f2)
	face := f3

	img := image.NewRGBA(image.Rect(0, 0, 1000, 5000))
	bounds := img.Bounds()

	str := ""
	for i := 0; i < 10; i++ {
		str += loremStr
	}

	d := &HSDrawer{Face: face, Str: str}
	ci := 3
	hwi := 15
	d.CursorIndex = &ci
	d.HighlightWordIndex = &hwi
	d.Selection = &loopers.SelectionIndexes{4, 50}
	c0 := DefaultColors
	d.Colors = &c0

	max := image.Point{bounds.Dx(), 100000}
	d.Measure(max)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Draw(img, &bounds)
	}
}
