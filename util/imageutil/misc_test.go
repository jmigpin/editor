package imageutil

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

var drawRect = image.Rect(0, 0, 400, 400)

func BenchmarkFillRect1(b *testing.B) {
	img := image.NewRGBA(drawRect)
	bounds := img.Bounds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FillRectangle(img, bounds, color.White)
	}
}
func BenchmarkFillRect2(b *testing.B) {
	img := NewBGRA(&drawRect)
	bounds := img.Bounds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FillRectangle(img, bounds, color.White)
	}
}
func BenchmarkDrawBGRA(b *testing.B) {
	img := NewBGRA(&drawRect)
	bounds := img.Bounds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src := image.NewUniform(color.White)
		draw.Draw(img, bounds, src, image.Point{}, draw.Src)
	}
}
