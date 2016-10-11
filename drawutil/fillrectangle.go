package drawutil

import (
	"image"
	"image/color"
	"image/draw"
	"sync"
)

var FillRectangle = FillRectangleLanesConc

func FillRectangleLanes(img draw.Image, r *image.Rectangle, col color.Color) {
	// faster
	srgba, ok := img.(interface {
		SetRGBA(int, int, color.RGBA)
	})
	if ok {
		c2 := color.RGBAModel.Convert(col).(color.RGBA)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				srgba.SetRGBA(x, y, c2)
			}
		}
		return
	}
	// common lane
	FillRectangleCommon(img, r, col)
}
func FillRectangleCommon(img draw.Image, r *image.Rectangle, col color.Color) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.Set(x, y, col)
		}
	}
}
func FillRectangleLanesConc(img draw.Image, r *image.Rectangle, col color.Color) {
	// faster lane
	srgba, ok := img.(interface {
		SetRGBA(int, int, color.RGBA)
	})
	if ok {
		c2 := color.RGBAModel.Convert(col).(color.RGBA)
		fillRectangleConcurrentFn(r, func(x, y int) {
			srgba.SetRGBA(x, y, c2)
		})
		return
	}
	// common lane
	FillRectangleCommonConc(img, r, col)
}
func FillRectangleCommonConc(img draw.Image, r *image.Rectangle, col color.Color) {
	fillRectangleConcurrentFn(r, func(x, y int) {
		img.Set(x, y, col)
	})
}

func fillRectangleConcurrentFn(r *image.Rectangle, fn func(x, y int)) {
	var wg sync.WaitGroup
	chunk := 64
	for y := r.Min.Y; y < r.Max.Y; y += chunk {
		my := y + chunk
		for x := r.Min.X; x < r.Max.X; x += chunk {
			mx := x + chunk
			r1 := image.Rect(x, y, mx, my).Intersect(*r)

			wg.Add(1)
			go func(r *image.Rectangle) {
				defer wg.Done()
				for y := r.Min.Y; y < r.Max.Y; y++ {
					for x := r.Min.X; x < r.Max.X; x++ {
						fn(x, y)
					}
				}
			}(&r1)
		}
	}
	wg.Wait()
}
