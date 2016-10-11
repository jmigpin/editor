package xutil

import "image/color"

// used by cursors
func ColorUint16s(c color.Color) (uint16, uint16, uint16, uint16) {
	r, g, b, a := c.RGBA()
	return uint16(r << 8), uint16(g << 8), uint16(b << 8), uint16(a)
}

//func RGBADataForX(img image.Image, r *image.Rectangle) []uint8 {
//// fast lane: RGBA
//rgbaImg, ok := img.(*image.RGBA)
//if ok {
//d := make([]uint8, r.Dx()*r.Dy()*4)
//i := 0
//for y := r.Min.Y; y < r.Max.Y; y++ {
//for x := r.Min.X; x < r.Max.X; x++ {
//c2 := rgbaImg.RGBAAt(x, y)
//// X wants BGRA
//d[i+0] = c2.B
//d[i+1] = c2.G
//d[i+2] = c2.R
//d[i+3] = c2.A

////// Faster, but its RGBA instead of the needed BGRA
////i2 := rgbaImg.PixOffset(x, y)
////copy(d[i:],rgbaImg.Pix[i2:i2+4])

//i += 4
//}
//}
//return d
//}
//// common lane
//d := make([]uint8, r.Dx()*r.Dy()*4)
//i := 0
//for y := r.Min.Y; y < r.Max.Y; y++ {
//for x := r.Min.X; x < r.Max.X; x++ {
//c1 := img.At(x, y)
//c2 := color.RGBAModel.Convert(c1).(color.RGBA)
//// X wants BGRA
//d[i+0] = c2.B
//d[i+1] = c2.G
//d[i+2] = c2.R
//d[i+3] = c2.A
//i += 4
//}
//}
//return d
//}

//func SendImageInTiles(gctx *GContext, x, y int, img image.Image, r *image.Rectangle) {
//var wg sync.WaitGroup
//chunk := 64
//for yi := r.Min.Y; yi < r.Max.Y; yi += chunk {
//my := yi + chunk
//for xi := r.Min.X; xi < r.Max.X; xi += chunk {
//mx := xi + chunk
//r2 := image.Rect(xi, yi, mx, my).Intersect(*r)
//wg.Add(1)
//go func(r2 image.Rectangle, xi, yi int) {
//defer wg.Done()
//data := RGBADataForX(img, &r2)
//w, h := r2.Dx(), r2.Dy()
//x0 := x + xi - r.Min.X
//y0 := y + yi - r.Min.Y
//gctx.PutImageData(x0, y0, w, h, data)
//}(r2, xi, yi)
//}
//}
//wg.Wait()
//}
