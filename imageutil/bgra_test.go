package imageutil

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func getBGRAImage() draw.Image {
	r := image.Rect(0, 0, 3000, 1000)
	return &BGRA{*image.NewRGBA(r)}
}

func BenchmarkFILanesConc(b *testing.B) {
	img := getBGRAImage()
	r := img.Bounds()
	FillRectangleLanesConc(img, &r, color.White)
}
func BenchmarkFICommon(b *testing.B) {
	img := getBGRAImage()
	r := img.Bounds()
	FillRectangleCommon(img, &r, color.White)
}
