package fontutil

import (
	_ "embed"
	"image"

	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"
)

// Embedded Noto Emoji variable font fallback.
// License: SIL Open Font License 1.1.
// See util/fontutil/data/OFL.txt.
//
//go:embed data/NotoEmoji-VariableFont_wght.ttf
var defaultEmojiTTF []byte

// Embedded Noto Sans Symbols 2 fallback.
// License: SIL Open Font License 1.1.
// See util/fontutil/data/OFL.txt.
//
//go:embed data/NotoSansSymbols2-Regular.ttf
var defaultSymbolsTTF []byte

func DefaultFont() *Font {
	return FontsMan.mustFont(goregular.TTF)
}
func DefaultMonoFont() *Font {
	return FontsMan.mustFont(gomono.TTF)
}
func DefaultEmojiFont() *Font {
	return FontsMan.mustFont(defaultEmojiTTF)
}
func DefaultSymbolsFont() *Font {
	return FontsMan.mustFont(defaultSymbolsTTF)
}

//----------

func DefaultFontFace() *FontFace {
	return DefaultFont().FontFace(DefaultFaceOptions)
}
func DefaultMonoFontFace() *FontFace {
	return DefaultMonoFont().FontFace(DefaultFaceOptions)
}

var DefaultFaceOptions = NewFaceOptions(12, 72)

//----------

func Rect266MinFloorMaxCeil(r fixed.Rectangle26_6) image.Rectangle {
	min := image.Point{r.Min.X.Floor(), r.Min.Y.Floor()}
	max := image.Point{r.Max.X.Ceil(), r.Max.Y.Ceil()}
	return image.Rectangle{min, max}
}

func Float32ToFixed266(v float32) fixed.Int26_6 {
	return fixed.Int26_6(v * 64)
}
func Float64ToFixed266(v float64) fixed.Int26_6 {
	return fixed.Int26_6(v * 64)
}
func Fixed266ToFloat64(v fixed.Int26_6) float64 {
	return float64(v) / float64(64)
}
