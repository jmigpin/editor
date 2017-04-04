package imageutil

import (
	"image"
	"image/color"
	"image/draw"
	"sync"
)

//var FillRectangle = FillRectangleLanesConc
var FillRectangle = FillRectangle5

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

// Image must be RGBA or will panic
func FillRectangle3(img0 draw.Image, r *image.Rectangle, c0 color.Color) {
	img, c := RGBAImageAndColor(img0, c0)
	u := []uint8{c.R, c.G, c.B, c.A}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			do := img.PixOffset(x, y)
			copy(img.Pix[do:], u)
		}
	}
}

// Image must be RGBA or will panic
func FillRectangle4(img0 draw.Image, r *image.Rectangle, c0 color.Color) {
	img, c := RGBAImageAndColor(img0, c0)
	u := []uint8{c.R, c.G, c.B, c.A}
	fillRectangleConcurrentFn(r, func(x, y int) {
		do := img.PixOffset(x, y)
		copy(img.Pix[do:], u)
	})
}

// Image must be RGBA or will panic
func FillRectangle5(img0 draw.Image, r *image.Rectangle, c0 color.Color) {
	img, c := RGBAImageAndColor(img0, c0)
	u := []uint8{c.R, c.G, c.B, c.A}
	// first line
	y := r.Min.Y
	if y < r.Max.Y {
		for x := r.Min.X; x < r.Max.X; x++ {
			do := img.PixOffset(x, y)
			copy(img.Pix[do:], u)
		}
	}
	// other lines
	x := r.Min.X
	if x < r.Max.X {
		so := img.PixOffset(x, r.Min.Y)
		for y := r.Min.Y + 1; y < r.Max.Y; y++ {
			do := img.PixOffset(x, y)
			copy(img.Pix[do:], img.Pix[so:so+r.Dx()*4])
		}
	}
}

func RGBAImageAndColor(img draw.Image, c color.Color) (*image.RGBA, *color.RGBA) {
	c2 := color.RGBAModel.Convert(c).(color.RGBA)
	// BGRA
	bgra, ok := img.(*BGRA)
	if ok {
		c2.R, c2.B = c2.B, c2.R
		return &bgra.RGBA, &c2
	}
	// RGBA
	rgba := img.(*image.RGBA)
	return rgba, &c2
}
