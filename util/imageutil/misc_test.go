package imageutil

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

var drawRect = image.Rect(0, 0, 400, 400)

func BenchmarkFillRect1(b *testing.B) {
	img := image.NewRGBA(drawRect)
	bounds := img.Bounds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FillRectangle(img, bounds, color.White)
	}
}
func BenchmarkFillRect2(b *testing.B) {
	img := NewBGRA(&drawRect)
	bounds := img.Bounds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FillRectangle(img, bounds, color.White)
	}
}
func BenchmarkDrawBGRA(b *testing.B) {
	img := NewBGRA(&drawRect)
	bounds := img.Bounds()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src := image.NewUniform(color.White)
		draw.Draw(img, bounds, src, image.Point{}, draw.Src)
	}
}

//----------

func TestComplimentRgba(t *testing.T) {
	v, expect := 0xff6699, 0x66ffcc
	c2 := RgbaFromInt(v)
	c3 := Complement(c2)
	v2 := RgbaToInt(c3)
	if v2 != expect {
		t.Fatalf("%06x exp=%06x got=%06x\n", v, expect, v2)
	}
}

func TestInvertRgba(t *testing.T) {
	v, expect := 0xff6699, 0x009966
	c2 := RgbaFromInt(v)
	c3 := Invert(c2)
	v2 := RgbaToInt(c3)
	if v2 != expect {
		t.Fatalf("%06x exp=%06x got=%06x\n", v, expect, v2)
	}
}

func TestLinearInvertRgba(t *testing.T) {
	fn := NewLinearInvertFn2(0.56, 2.5) // match gimp results
	w := []int{
		0xeaffff, 0x750000,
		0xffffea, 0x000075,
	}
	for i := 0; i < len(w); i += 2 {
		v, exp := w[i], w[i+1]
		c2 := RgbaFromInt(v)
		c3 := fn(c2)
		v2 := RgbaToInt(RgbaColor(c3))
		if v2 != exp {
			t.Fatalf("%06x exp=%06x got=%06x\n", v, exp, v2)
		}
	}
}
