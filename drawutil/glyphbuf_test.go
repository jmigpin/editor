package drawutil

import (
	"io/ioutil"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/golang/freetype/truetype"
)

var font0Filename = "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"
var loremText = `Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`

var dpi = float64(72)
var size = float64(12)
var scale = fixed.Int26_6(0.5 + (size * dpi * 64 / 72))
var font0 = initFont()

func initFont() *truetype.Font {
	bytes, err := ioutil.ReadFile(font0Filename)
	if err != nil {
		panic(err)
	}
	f, err := truetype.Parse(bytes)
	if err != nil {
		panic(err)
	}
	return f
}

func BenchmarkF0(b *testing.B) {
	var glyphBuf truetype.GlyphBuf
	for i := 0; i < 300; i++ {
		for _, ru := range loremText {
			if err := glyphBuf.Load(font0, scale, font0.Index(ru), font.HintingFull); err != nil {
				b.Fatal(err)
			}
		}
	}
}
func BenchmarkF1(b *testing.B) {
	m := make(map[rune]*truetype.GlyphBuf)
	for i := 0; i < 300; i++ {
		for _, ru := range loremText {
			g, ok := m[ru]
			if !ok {
				var glyphBuf truetype.GlyphBuf
				if err := glyphBuf.Load(font0, scale, font0.Index(ru), font.HintingFull); err != nil {
					b.Fatal(err)
				}
				m[ru] = &glyphBuf
			}
			_ = g
		}
	}
}
