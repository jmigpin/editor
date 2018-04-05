package widget

import (
	"image/color"

	"github.com/golang/freetype/truetype"
	"github.com/jmigpin/editor/util/drawutil"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
)

var (
	White color.Color = color.RGBA{255, 255, 255, 255}
	Black color.Color = color.RGBA{0, 0, 0, 255}

	// used if a color name is not found
	defaultThemeColor color.Color = color.RGBA{255, 255, 0, 255} // yellow
)

//----------

// nil is a valid receiver.
type Theme struct {
	Font    ThemeFont
	Palette Palette // Note: EmbedNode.ThemePalette() checks if theme is nil
}

func (t *Theme) empty() bool {
	return t == nil || (t.Font == nil && (t.Palette == nil || t.Palette.Empty()))
}

func (t *Theme) Copy() *Theme {
	if t == nil {
		return &Theme{Palette: MakePalette()}
	}
	u := *t
	u.Palette = t.Palette.Copy()
	return &u
}

//----------

// nil is a valid receiver.
type Palette map[string]color.Color

func MakePalette() Palette {
	return make(Palette)
}

func (pal Palette) Empty() bool {
	return pal == nil || len(pal) == 0
}

func (pal Palette) Copy() Palette {
	pal2 := MakePalette()
	for k, v := range pal {
		pal2[k] = v
	}
	return pal2
}

//----------

var defaultPalette = Palette{
	"fg": Black,
	"bg": White,
}

//----------

func TreeThemePaletteColor(name string, en *EmbedNode) color.Color {
	for n := en; n != nil; n = n.parent {
		if n.theme != nil && n.theme.Palette != nil {
			if c, ok := n.theme.Palette[name]; ok {
				return c
			}
		}
	}
	if c, ok := defaultPalette[name]; ok {
		return c
	}
	return defaultThemeColor
}

func TreeThemeFont(en *EmbedNode) ThemeFont {
	for n := en; n != nil; n = n.parent {
		if n.theme != nil && n.theme.Font != nil {
			return n.theme.Font
		}
	}
	return defaultThemeFont()
}

func ThemeFontOrDefault(t *Theme) ThemeFont {
	if t != nil && t.Font != nil {
		return t.Font
	}
	return defaultThemeFont()
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

//----------

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

//----------

var _dft ThemeFont

func defaultThemeFont() ThemeFont {
	if _dft == nil {
		_dft = goregularThemeFont()
	}
	return _dft
}

func goregularThemeFont() *TTThemeFont {
	opt := &truetype.Options{}
	tf, err := NewTTThemeFont(goregular.TTF, opt)
	if err != nil {
		panic(err)
	}
	return tf
}
