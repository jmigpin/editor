package widget

import (
	"image/color"

	"github.com/jmigpin/editor/util/fontutil"
	"github.com/jmigpin/editor/util/imageutil"
)

//----------

var DefaultPalette = Palette{
	"text_cursor_fg":             nil, // present but nil uses the current fg
	"text_fg":                    cint(0x0),
	"text_bg":                    cint(0xffffff),
	"text_selection_fg":          nil,
	"text_selection_bg":          cint(0xeeee9e), // yellow
	"text_colorize_string_fg":    cint(0x008b00), // green
	"text_colorize_string_bg":    nil,
	"text_colorize_comments_fg":  cint(0x757575), // grey 600
	"text_colorize_comments_bg":  nil,
	"text_highlightword_fg":      nil,
	"text_highlightword_bg":      cint(0xc6ee9e), // green
	"text_wrapline_fg":           cint(0x0),
	"text_wrapline_bg":           cint(0xd8d8d8),
	"text_parenthesis_fg":        cint(0x0),
	"text_parenthesis_bg":        cint(0xc3c3c3),
	"text_annotations_fg":        cint(0x0),
	"text_annotations_bg":        cint(0xb0e0ef),
	"text_annotations_select_fg": cint(0x0),
	"text_annotations_select_bg": cint(0xefc7b0),

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
	FontFace          *fontutil.FontFace
	Palette           Palette
	PaletteNamePrefix string
}

func (t *Theme) empty() bool {
	return (t.FontFace == nil &&
		(t.Palette == nil || t.Palette.Empty()) &&
		t.PaletteNamePrefix == "")
}

func (t *Theme) ClearIfEmpty() {
	if t.empty() {
		*t = Theme{}
	}
}

//----------

// Can be set to nil to erase.
func (t *Theme) SetFontFace(ff *fontutil.FontFace) {
	t.FontFace = ff
	t.ClearIfEmpty()
}

// Can be set to nil to erase.
func (t *Theme) SetPalette(p Palette) {
	t.Palette = p
	t.ClearIfEmpty()
}

// Can be set to nil to erase.
func (t *Theme) SetPaletteColor(name string, c color.Color) {
	// delete color
	if c == nil {
		if t.Palette != nil {
			delete(t.Palette, name)
		}
		t.ClearIfEmpty()
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
	t.ClearIfEmpty()
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

func cint(c int) color.RGBA {
	return imageutil.RgbaFromInt(c)
}
