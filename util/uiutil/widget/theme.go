package widget

import (
	"image/color"
	"log"

	"github.com/golang/freetype/truetype"
	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/imageutil"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
)

//----------

var DefaultPalette = Palette{
	"text_cursor_fg":            nil, // present but nil uses the current fg
	"text_fg":                   cint(0x0),
	"text_bg":                   cint(0xffffff),
	"text_selection_fg":         nil,
	"text_selection_bg":         cint(0xeeee9e), // yellow
	"text_colorize_string_fg":   cint(0x008b00), // green
	"text_colorize_comments_fg": cint(0x757575), // grey 600
	"text_highlightword_fg":     nil,
	"text_highlightword_bg":     cint(0xc6ee9e), // green
	"text_wrapline_fg":          cint(0x0),
	"text_wrapline_bg":          cint(0xd8d8d8),
	"text_parenthesis_fg":       cint(0x0),
	"text_parenthesis_bg":       cint(0xc3c3c3),

	"scrollbar_bg":        cint(0xf2f2f2),
	"scrollhandle_normal": cint(0xb2b2b2),
	"scrollhandle_hover":  cint(0x8e8e8e),
	"scrollhandle_select": cint(0x5f5f5f),

	"button_hover_fg":  nil,
	"button_hover_bg":  cint(0xdddddd),
	"button_down_fg":   nil,
	"button_down_bg":   cint(0xaaaaaa),
	"button_sticky_fg": cint(0xffffff),
	"button_sticky_bg": cint(0x0),

	"pad":    cint(0x8080ff), // helpful color to debug
	"border": cint(0x00ff00), // helpful color to debug
	"rect":   cint(0xff8000), // helpful color to debug
}

//----------

type Theme struct {
	Font              ThemeFont
	Palette           Palette
	PaletteNamePrefix string
}

func (t *Theme) empty() bool {
	return (t.Font == nil &&
		(t.Palette == nil || t.Palette.Empty()) &&
		t.PaletteNamePrefix == "")
}

func (t *Theme) Clear() {
	if t.empty() {
		*t = Theme{}
	}
}

//----------

// Can be set to nil to erase.
func (t *Theme) SetFont(f ThemeFont) {
	t.Font = f
	t.Clear()
}

// Can be set to nil to erase.
func (t *Theme) SetPalette(p Palette) {
	t.Palette = p
	t.Clear()
}

// Can be set to nil to erase.
func (t *Theme) SetPaletteColor(name string, c color.Color) {
	// delete color
	if c == nil {
		if t.Palette != nil {
			delete(t.Palette, name)
		}
		t.Clear()
		return
	}

	if t.Palette == nil {
		t.Palette = Palette{}
	}
	t.Palette[name] = c
}

// Can be set to "" to erase.
func (t *Theme) SetPaletteNamePrefix(prefix string) {
	t.PaletteNamePrefix = prefix
	t.Clear()
}

//----------

type Palette map[string]color.Color

func (pal Palette) Empty() bool {
	return pal == nil || len(pal) == 0
}

func (pal Palette) Merge(p2 Palette) {
	for k, v := range p2 {
		pal[k] = v
	}
}

//----------

type ThemeFont interface {
	Face(*ThemeFontOptions) font.Face
	CloseFaces()
}

type ThemeFontOptions struct {
	Size ThemeFontOptionsSize
}

type ThemeFontOptionsSize int

const (
	TFOSNormal ThemeFontOptionsSize = iota // default
	TFOSSmall
)

//----------

// Truetype theme font.
type TTThemeFont struct {
	ttfont *truetype.Font
	opt    *truetype.Options
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
		faces:  map[truetype.Options]font.Face{},
	}
	return tf, nil
}

func (tf *TTThemeFont) Face(ffopt *ThemeFontOptions) font.Face {
	opt2 := *tf.opt
	if ffopt != nil {
		if ffopt.Size == TFOSSmall {
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

func (tf *TTThemeFont) CloseFaces() {
	for _, f := range tf.faces {
		err := f.Close()
		if err != nil {
			log.Print(err)
		}
	}
	tf.faces = map[truetype.Options]font.Face{}
}

//----------

var _dft ThemeFont

func DefaultThemeFont() ThemeFont {
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

//----------

func cint(c int) color.RGBA {
	return imageutil.IntRGBA(c)
}
