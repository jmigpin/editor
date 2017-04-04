package imageutil

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func getImage() draw.Image {
	r := image.Rect(0, 0, 300, 300)
	return image.NewRGBA(r)
}

func BenchmarkFR(b *testing.B) {
	img := getImage().(*image.RGBA)
	r := img.Bounds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FillRectangle(img, &r, &color.White)
	}
}
