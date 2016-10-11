package drawutil

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func getImage() draw.Image {
	r := image.Rect(0, 0, 3000, 1500)
	//r := image.Rect(0, 0, 1200, 800)
	return image.NewRGBA(r)
}

func BenchmarkFR0(b *testing.B) {
	img := getImage()
	r := img.Bounds()
	FillRectangleLanes(img, &r, color.White)
}
func BenchmarkFR1(b *testing.B) {
	img := getImage()
	r := img.Bounds()
	FillRectangleCommon(img, &r, color.White)
}
func BenchmarkFR2(b *testing.B) {
	img := getImage()
	r := img.Bounds()
	FillRectangleLanesConc(img, &r, color.White)
}
func BenchmarkFR3(b *testing.B) {
	img := getImage()
	r := img.Bounds()
	FillRectangleCommonConc(img, &r, color.White)
}
