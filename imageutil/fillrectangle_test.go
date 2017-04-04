package imageutil

// TEST
// go test -bench  "FRLanes.*|FR3|FR4" -cpu 4
// go test -bench  "FRLanes.*|FR3|FR4|FR5"

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

func BenchmarkFR0Lanes(b *testing.B) {
	img := getImage()
	r := img.Bounds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FillRectangleLanes(img, &r, color.White)
	}
}
func BenchmarkFRCommon(b *testing.B) {
	img := getImage()
	r := img.Bounds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FillRectangleCommon(img, &r, color.White)
	}
}
func BenchmarkFRLanesConc(b *testing.B) {
	img := getImage()
	r := img.Bounds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FillRectangleLanesConc(img, &r, color.White)
	}
}
func BenchmarkFRCommonConc(b *testing.B) {
	img := getImage()
	r := img.Bounds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FillRectangleCommonConc(img, &r, color.White)
	}
}

func BenchmarkFR3(b *testing.B) {
	img := getImage().(*image.RGBA)
	r := img.Bounds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FillRectangle3(img, &r, &color.RGBA{0xff, 0xff, 0xff, 0xff})
	}
}
func BenchmarkFR4(b *testing.B) {
	img := getImage().(*image.RGBA)
	r := img.Bounds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FillRectangle4(img, &r, &color.RGBA{0xff, 0xff, 0xff, 0xff})
	}
}
func BenchmarkFR5(b *testing.B) {
	img := getImage().(*image.RGBA)
	r := img.Bounds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FillRectangle5(img, &r, &color.RGBA{0xff, 0xff, 0xff, 0xff})
	}
}
