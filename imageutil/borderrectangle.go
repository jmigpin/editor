package imageutil

import (
	"image"
	"image/color"
	"image/draw"
)

func BorderRectangle(img0 draw.Image, r *image.Rectangle, c0 color.Color, size int) {
	img, c := RGBAImageAndColor(img0, c0)
	u := image.NewUniform(c)
	var sr [4]image.Rectangle
	// top
	sr[0] = *r
	sr[0].Max.Y = r.Min.Y + size
	// bottom
	sr[1] = *r
	sr[1].Min.Y = r.Max.Y - size
	// left
	sr[2] = *r
	sr[2].Max.X = r.Min.X + size
	sr[2].Min.Y = r.Min.Y + size
	sr[2].Max.Y = r.Max.Y - size
	// right
	sr[3] = *r
	sr[3].Min.X = r.Max.X - size
	sr[3].Min.Y = r.Min.Y + size
	sr[3].Max.Y = r.Max.Y - size
	for _, r2 := range sr {
		r2 = r2.Intersect(*r)
		draw.Draw(img, r2, u, image.Point{}, draw.Src)
	}
}
