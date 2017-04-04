package imageutil

import (
	"image"
	"image/draw"
)

func Copy(dst, src draw.Image, r *image.Rectangle) {
	draw.Draw(dst, *r, src, image.Point{}, draw.Src)
}
