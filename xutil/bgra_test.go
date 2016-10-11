package xutil

import (
	"image"
	"image/color"
	"image/draw"
	"jmigpin/editor/drawutil"
	"testing"
)

func getImage() draw.Image {
	r := image.Rect(0, 0, 3000, 1000)
	//return image.NewRGBA(r)
	return &BGRA{*image.NewRGBA(r)}
}

func BenchmarkFR0(b *testing.B) {
	img := getImage()
	r := img.Bounds()
	drawutil.FillRectangle(img, &r, color.White)
}
func BenchmarkFR1(b *testing.B) {
	img := getImage()
	r := img.Bounds()
	drawutil.FillRectangleCommon(img, &r, color.White)
}
