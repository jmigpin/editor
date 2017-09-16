package drawutil2

import (
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
)

func NewFace(font *truetype.Font, opt *truetype.Options) font.Face {
	face := truetype.NewFace(font, opt)
	f2 := NewFaceRunes(face)
	f3 := NewFaceCache(f2)
	return f3
}

func GetTestFace() font.Face {
	ttf := goregular.TTF
	f, err := truetype.Parse(ttf)
	if err != nil {
		panic(err)
	}
	ttOpt := &truetype.Options{Size: 12, Hinting: font.HintingFull}
	return NewFace(f, ttOpt)
}
