package loopers

import (
	"image"
	"image/color"
	"testing"

	"github.com/jmigpin/editor/drawutil2"
)

var loremStr = `Lorem ipsum dolor sit amet, consectetur adlpiscing elit, sed do eiusmod tempor incidldunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`

func BenchmarkDraw(b *testing.B) {
	face := drawutil2.GetTestFace()

	img := image.NewRGBA(image.Rect(0, 0, 50000, 200))
	bounds := img.Bounds()
	strl := NewStringLooper(face, loremStr)
	dl := NewDrawLooper(strl, img, &bounds)
	dl.Fg = color.Black
	dl.Looper = strl

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dl.Loop(func() bool { return true })
	}
}
