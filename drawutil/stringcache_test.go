package drawutil

import (
	"testing"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/goregular"
)

var str = ""

func init() {
	for i := 0; i < 100000; i++ {
		str += "0123456789\n"
	}
}

func BenchmarkCalcRuneData0(t *testing.B) {
	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return
	}
	opt := &truetype.Options{Size: 13}
	face := NewFace(font, opt)
	sc := NewStringCache(face)
	w := 50
	sc.CalcRuneData(str, w)
}
