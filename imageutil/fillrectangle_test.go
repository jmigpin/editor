package imageutil

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func getImage() draw.Image {
	r := image.Rect(0, 0, 3000, 1500)
	return image.NewRGBA(r)
}

func BenchmarkFR0Lanes(b *testing.B) {
	img := getImage()
	r := img.Bounds()
	FillRectangleLanes(img, &r, color.White)
}
func BenchmarkFRCommon(b *testing.B) {
	img := getImage()
	r := img.Bounds()
	FillRectangleCommon(img, &r, color.White)
}
func BenchmarkFRLanesConc(b *testing.B) {
	img := getImage()
	r := img.Bounds()
	FillRectangleLanesConc(img, &r, color.White)
}
func BenchmarkFRCommonConc(b *testing.B) {
	img := getImage()
	r := img.Bounds()
	FillRectangleCommonConc(img, &r, color.White)
}
