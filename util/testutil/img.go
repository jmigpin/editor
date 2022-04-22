package testutil

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	//_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"golang.org/x/image/colornames"
)

func OpenImage(filename string) (image.Image, string, error) {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	return image.Decode(f)
}

//----------

func ClearImg(img draw.Image) {
	ClearImg2(img, colornames.Lightgray)
}
func ClearImg2(img draw.Image, c color.Color) {
	r := img.Bounds()
	src := image.NewUniform(c)
	draw.DrawMask(img, r, src, image.Point{}, nil, image.Point{}, draw.Src)
}

func GenerateImg(r image.Rectangle, seed int) image.Image {
	img := image.NewRGBA(r)
	for y := 0; y < img.Bounds().Max.Y; y++ {
		for x := 0; x < img.Bounds().Max.X; x++ {
			v := byte(255 - seed*x*y)
			c := color.RGBA{v, v, 0, 255}
			img.Set(x, y, c)
		}
	}
	return img
}

//----------

func SPrintImg(img image.Image) string {
	//b, _ := imageutil.EncodeToSixel(img)
	//return string(b)
	return "TODO:encodetosixel"
}

func SPrintImgs(imgs ...image.Image) string {
	s := ""
	for i, img := range imgs {
		if i > 0 {
			s += " "
		}
		s += SPrintImg(img)
	}
	return s
}

//----------

func CompareImgs(img1, img2 image.Image) error {
	if img1.Bounds() != img2.Bounds() {
		return fmt.Errorf("bounds: %v %v", img1.Bounds(), img2.Bounds())
	}
	b1 := img1.Bounds()
	nFails := 0
	firstFail := image.Point{}
	for y := b1.Min.Y; y < b1.Max.Y; y++ {
		for x := b1.Min.X; x < b1.Max.X; x++ {
			c1 := color.RGBAModel.Convert(img1.At(x, y))
			c2 := color.RGBAModel.Convert(img2.At(x, y))
			if c1 != c2 {
				nFails++
				if nFails == 1 {
					firstFail = image.Point{x, y}
				}
			}
		}
	}
	if nFails > 0 {
		x, y := firstFail.X, firstFail.Y
		c1 := color.RGBAModel.Convert(img1.At(x, y))
		c2 := color.RGBAModel.Convert(img2.At(x, y))
		return fmt.Errorf("colors: xy=(%v,%v): %v %v (nfails: %v)", x, y, c1, c2, nFails)
	}
	return nil
}

func CompareImgsOrSavePng(img1 image.Image, filename2 string) error {
	f2, err := os.Open(filename2)
	if err != nil {
		if os.IsNotExist(err) {
			// save image
			f3, err := os.Create(filename2)
			if err != nil {
				return err
			}
			defer f3.Close()
			if err := png.Encode(f3, img1); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	defer f2.Close()
	img2, _, err := image.Decode(f2)
	if err != nil {
		return err
	}

	err = CompareImgs(img1, img2)
	if err != nil {
		return fmt.Errorf("%w:\ngot:\n%s\nexpected:\n%v", err, SPrintImg(img1), SPrintImg(img2))
	}
	return nil
}

//----------

func DrawPoint(img draw.Image, p image.Point, size int, c color.Color) {
	r := image.Rect(p.X, p.Y, p.X+size, p.Y+size)
	r2 := r.Intersect(img.Bounds())
	src := image.NewUniform(c)
	draw.Draw(img, r2, src, image.Point{}, draw.Src)
}
