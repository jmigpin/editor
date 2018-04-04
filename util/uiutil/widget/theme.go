package widget

import (
	"image/color"

	"github.com/golang/freetype/truetype"
	"github.com/jmigpin/editor/util/drawutil"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
)

var (
	White   color.Color = color.RGBA{255, 255, 255, 255}
	Black   color.Color = color.RGBA{0, 0, 0, 255}
	NoColor color.Color = color.RGBA{255, 255, 0, 255}
)

//----------

// The nil-value is a valid receiver.
type Theme struct {
	font    ThemeFont
	palette Palette
}

func (t *Theme) Palette() Palette {
	if t == nil || t.palette == nil {
		return defaultPalette
	}
	return t.palette
}
func (t *Theme) PaletteCopy() Palette {
	pal := MakePalette()
	for k, v := range t.Palette() {
		pal[k] = v
	}
	return pal
}
func (t *Theme) SetPalette(p Palette) {
	t.palette = p
}

func (t *Theme) Font() ThemeFont {
	if t == nil || t.font == nil {
		return defaultThemeFont()
	}
	return t.font
}
func (t *Theme) SetFont(tf ThemeFont) {
	t.font = tf
}

//----------

type Palette map[string]color.Color

func MakePalette() Palette {
	return make(Palette)
}
func (pal Palette) Get(name string) color.Color {
	if v, ok := pal[name]; ok {
		return v
	}
	return NoColor
}
func (pal Palette) GetFrom(pal2 Palette, name string) {
	pal[name] = pal2[name]
}

var defaultPalette = Palette{
	"fg": Black,
	"bg": White,
}

//----------

type ThemeFont interface {
	Face(*ThemeFontOptions) font.Face
	Clear() // clears internal faces
}

type ThemeFontOptions struct {
	Size ThemeFontOptionsSize
}

type ThemeFontOptionsSize int

const (
	NormalTFOS ThemeFontOptionsSize = iota // default
	SmallTFOS
)

// Truetype theme font.
type TTThemeFont struct {
	opt    *truetype.Options
	ttfont *truetype.Font
	faces  map[truetype.Options]font.Face
}

func NewTTThemeFont(ttf []byte, opt *truetype.Options) (*TTThemeFont, error) {
	ttfont, err := truetype.Parse(ttf)
	if err != nil {
		return nil, err
	}
	tf := &TTThemeFont{
		opt:    opt,
		ttfont: ttfont,
		faces:  make(map[truetype.Options]font.Face),
	}
	return tf, nil
}
func (tf *TTThemeFont) Face(ffopt *ThemeFontOptions) font.Face {
	opt2 := *tf.opt
	if ffopt != nil {
		if ffopt.Size == SmallTFOS {
			opt2.Size *= float64(2) / 3
		}
	}
	face, ok := tf.faces[opt2]
	if !ok {
		face = drawutil.NewFace(tf.ttfont, &opt2)
		tf.faces[opt2] = face
	}
	return face
}

func (tf *TTThemeFont) Clear() {
	for _, f := range tf.faces {
		_ = f.Close()
	}
	tf.faces = make(map[truetype.Options]font.Face)
}

var _defaultThemeFont ThemeFont

func defaultThemeFont() ThemeFont {
	if _defaultThemeFont == nil {
		_defaultThemeFont = goregularThemeFont()
	}
	return _defaultThemeFont
}

func goregularThemeFont() *TTThemeFont {
	opt := &truetype.Options{}
	tf, err := NewTTThemeFont(goregular.TTF, opt)
	if err != nil {
		panic(err)
	}
	return tf
}
