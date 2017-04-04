package imageutil

import "image"

func CopyRGBA(dst, src *image.RGBA, r *image.Rectangle) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			do := dst.PixOffset(x, y)
			so := src.PixOffset(x, y)
			copy(dst.Pix[do:], src.Pix[so:so+4])
		}
	}
}
