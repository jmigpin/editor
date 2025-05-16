package ui

import (
	"fmt"
	"image"
	"image/color"
	"io/ioutil"

	"github.com/jmigpin/editor/util/fontutil"
	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/uiutil/widget"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

var ScrollBarLeft = true
var ScrollBarWidth int = 0 // 0=based on a portion of the font size

var TextAreaCommentsColor color.Color
var TextAreaStringsColor color.Color

const separatorWidth = 1 // col/row separators width

//----------

func lightThemeColors(node widget.Node) {
	pal := lightThemeColorsPal()
	pal.Merge(rowSquarePalette())
	pal.Merge(userPalette())
	node.Embed().SetThemePalette(pal)
}
func lightThemeColorsPal() widget.Palette {
	pal := widget.Palette{
		"text_cursor_fg":            cint(0x0),
		"text_fg":                   cint(0x0),
		"text_bg":                   cint(0xffffff),
		"text_selection_fg":         nil,
		"text_selection_bg":         cint(0xeeee9e), // yellow
		"text_colorize_string_fg":   cint(0x8b0000), // red
		"text_colorize_comments_fg": cint(0x008b00), // green
		"text_highlightword_fg":     nil,
		"text_highlightword_bg":     cint(0xc6ee9e), // green
		"text_wrapline_fg":          cint(0x0),
		"text_wrapline_bg":          cint(0xd8d8d8),
		"text_parenthesis_fg":       nil,
		"text_parenthesis_bg":       cint(0xd8d8d8),

		"toolbar_text_bg":          cint(0xecf0f1), // "clouds" grey
		"toolbar_text_wrapline_bg": cint(0xccccd8),

		"scrollbar_bg":        cint(0xf2f2f2),
		"scrollhandle_normal": imageutil.Shade(cint(0xf2f2f2), 0.20),
		"scrollhandle_hover":  imageutil.Shade(cint(0xf2f2f2), 0.30),
		"scrollhandle_select": imageutil.Shade(cint(0xf2f2f2), 0.40),

		"column_norows_rect":  cint(0xffffff),
		"columns_nocols_rect": cint(0xffffff),
		"colseparator_rect":   cint(0x0),
		"rowseparator_rect":   cint(0x0),
		"shadowsep_rect":      cint(0x0),

		"columnsquare": cint(0xccccd8),
		"rowsquare":    cint(0xccccd8),

		"mm_text_bg":          cint(0xecf0f1),
		"mm_button_hover_bg":  cint(0xcccccc),
		"mm_button_down_bg":   cint(0xbbbbbb),
		"mm_button_sticky_fg": cint(0xffffff),
		"mm_button_sticky_bg": cint(0x0),
		"mm_border":           cint(0x0),
		"mm_content_pad":      cint(0xecf0f1),
		"mm_content_border":   cint(0x0),

		"contextfloatbox_border": cint(0x0),
	}
	pal.Merge(rowSquarePalette())
	pal.Merge(userPalette())
	return pal
}

//----------

func acmeThemeColors(node widget.Node) {
	pal := acmeThemeColorsPal()
	pal.Merge(rowSquarePalette())
	pal.Merge(userPalette())
	node.Embed().SetThemePalette(pal)
}
func acmeThemeColorsPal() widget.Palette {
	pal := widget.Palette{
		"text_cursor_fg":            cint(0x0),
		"text_fg":                   cint(0x0),
		"text_bg":                   cint(0xffffea),
		"text_selection_fg":         nil,
		"text_selection_bg":         cint(0xeeee9e), // yellow
		"text_colorize_string_fg":   cint(0x8b0000), // red
		"text_colorize_comments_fg": cint(0x007500), // green
		"text_highlightword_fg":     nil,
		"text_highlightword_bg":     cint(0xc6ee9e), // green
		"text_wrapline_fg":          cint(0x0),
		"text_wrapline_bg":          cint(0xd8d8c6),

		"toolbar_text_bg":          cint(0xeaffff),
		"toolbar_text_wrapline_bg": cint(0xc6d8d8),

		"scrollbar_bg":        cint(0xf2f2de),
		"scrollhandle_normal": cint(0xc1c193),
		"scrollhandle_hover":  cint(0xadad6f),
		"scrollhandle_select": cint(0x99994c),

		"column_norows_rect":  cint(0xffffea),
		"columns_nocols_rect": cint(0xffffff),
		"colseparator_rect":   cint(0x0),
		"rowseparator_rect":   cint(0x0),
		"shadowsep_rect":      cint(0x0),

		"columnsquare": cint(0xc6d8d8),
		"rowsquare":    cint(0xc6d8d8),

		"mm_text_bg":          cint(0xeaffff),
		"mm_button_hover_bg":  imageutil.Shade(cint(0xeaffff), 0.10),
		"mm_button_down_bg":   imageutil.Shade(cint(0xeaffff), 0.20),
		"mm_button_sticky_bg": imageutil.Shade(cint(0xeaffff), 0.40),
		"mm_border":           cint(0x0),
		"mm_content_pad":      cint(0xeaffff),
		"mm_content_border":   cint(0x0),

		"contextfloatbox_border": cint(0x0),
	}
	pal.Merge(rowSquarePalette())
	pal.Merge(userPalette())
	return pal
}

//----------

func lightInvertedThemeColors(node widget.Node) {
	fn := newLinearInvertFn()
	pal := lightThemeColorsPal()
	for k, c := range pal {
		if c != nil {
			pal[k] = fn(c)
		}
	}
	pal.Merge(rowSquarePalette())
	pal.Merge(userPalette())
	node.Embed().SetThemePalette(pal)
}

//----------

func acmeInvertedThemeColors(node widget.Node) {
	fn := newLinearInvertFn()
	pal := acmeThemeColorsPal()
	for k, c := range pal {
		if c != nil {
			pal[k] = fn(c)
		}
	}
	pal.Merge(rowSquarePalette())
	pal.Merge(userPalette())
	node.Embed().SetThemePalette(pal)
}

//----------
//----------
//----------

// Palette with user supplied color options that should override themes.
func userPalette() widget.Palette {
	pal := widget.Palette{}

	setup := func(name string, c color.Color) {
		// not defined, nothing to change, use defaults
		if c == nil {
			return
		}
		// allow explicit setup to nil with a value of 0x1
		v := imageutil.RgbaToInt(imageutil.RgbaColor(c))
		if v == 0x1 {
			c = nil
		}

		pal[name] = c
	}

	setup("text_colorize_string_fg", TextAreaStringsColor)
	setup("text_colorize_comments_fg", TextAreaCommentsColor)
	return pal
}

//----------

func rowSquarePalette() widget.Palette {
	pal := widget.Palette{
		"rs_active":              cint(0x0),
		"rs_executing":           cint(0x0fad00),                       // dark green
		"rs_edited":              cint(0x0000ff),                       // blue
		"rs_disk_changes":        cint(0xff0000),                       // red
		"rs_not_exist":           cint(0xff9900),                       // orange
		"rs_duplicate":           cint(0x8888cc),                       // blueish
		"rs_duplicate_highlight": cint(0xffff00),                       // yellow
		"rs_annotations":         cint(0xd35400),                       // pumpkin
		"rs_annotations_edited":  imageutil.Tint(cint(0xd35400), 0.45), // pumpkin (brighter)
	}
	return pal
}

//----------
//----------
//----------

var ColorThemeCycler cycler = cycler{
	entries: []cycleEntry{
		{"light", lightThemeColors},
		{"acme", acmeThemeColors},
		{"lightInverted", lightInvertedThemeColors},
		{"acmeInverted", acmeInvertedThemeColors},
	},
}

//----------

var FontThemeCycler cycler = cycler{
	entries: []cycleEntry{
		{"regular", regularThemeFont},
		{"medium", mediumThemeFont},
		{"mono", monoThemeFont},
	},
}

//----------

func regularThemeFont(node widget.Node) {
	loadThemeFont("regular", node)
}
func mediumThemeFont(node widget.Node) {
	loadThemeFont("medium", node)
}
func monoThemeFont(node widget.Node) {
	loadThemeFont("mono", node)
}

//----------

func AddUserFont(filename string) error {
	// test now if it will load when needed
	_, err := ThemeFontFace(filename)
	if err != nil {
		return err
	}

	// prepare callback and add to font cycler
	f := func(node widget.Node) {
		_ = loadThemeFont(filename, node)
	}
	e := cycleEntry{filename, f}
	FontThemeCycler.entries = append(FontThemeCycler.entries, e)
	FontThemeCycler.CurName = filename
	return nil
}

//----------

func loadThemeFont(name string, node widget.Node) error {
	// close previous faces
	ff0 := node.Embed().TreeThemeFontFace()
	ff0.Font.ClearFacesCache()

	ff, err := ThemeFontFace(name)
	if err != nil {
		return err
	}
	node.Embed().SetThemeFontFace(ff)
	return nil
}

//----------

var FontFaceOptions opentype.FaceOptions

func ThemeFontFace(name string) (*fontutil.FontFace, error) {
	return ThemeFontFace2(name, 0)
}
func ThemeFontFace2(name string, size float64) (*fontutil.FontFace, error) {
	b, err := fontBytes(name)
	if err != nil {
		return nil, err
	}
	f, err := fontutil.FontsMan.Font(b)
	if err != nil {
		return nil, err
	}
	opt := FontFaceOptions // copy
	if size != 0 {
		opt.Size = size
	}
	return f.FontFace(opt), nil
}

func fontBytes(name string) ([]byte, error) {
	switch name {
	case "regular":
		return goregular.TTF, nil
	case "medium":
		return gomedium.TTF, nil
	case "mono":
		return gomono.TTF, nil
	default:
		return ioutil.ReadFile(name)
	}
}

//----------

type cycler struct {
	CurName string
	entries []cycleEntry
}

func (c *cycler) GetIndex(name string) (int, bool) {
	for i, e := range c.entries {
		if e.name == name {
			return i, true
		}
	}
	return -1, false
}

func (c *cycler) Cycle(node widget.Node) {
	i := 0
	if c.CurName != "" {
		k, ok := c.GetIndex(c.CurName)
		if !ok {
			panic(fmt.Sprintf("cycle name not found: %v", c.CurName))
		}
		i = (k + 1) % len(c.entries)
	}
	c.Set(c.entries[i].name, node)
}

func (c *cycler) Set(name string, node widget.Node) {
	i, ok := c.GetIndex(name)
	if !ok {
		panic(fmt.Sprintf("cycle name not found: %v", name))
	}
	c.CurName = name
	c.entries[i].fn(node)
}

func (c *cycler) Names() []string {
	w := []string{}
	for _, e := range c.entries {
		w = append(w, e.name)
	}
	return w
}

//----------

type cycleEntry struct {
	name string
	fn   func(widget.Node)
}

//----------

var UIThemeUtil uiThemeUtil

type uiThemeUtil struct{}

func (uitu *uiThemeUtil) RowMinimumHeight(ff *fontutil.FontFace) int {
	return ff.LineHeightInt()
}
func (uitu *uiThemeUtil) RowSquareSize(ff *fontutil.FontFace) image.Point {
	lh := ff.LineHeightFloat()
	w := int(lh * 3 / 4)
	return image.Point{w, int(lh)}
}

func (uitu *uiThemeUtil) GetScrollBarWidth(ff *fontutil.FontFace) int {
	if ScrollBarWidth != 0 {
		return ScrollBarWidth
	}
	lh := ff.LineHeightFloat()
	w := int(lh * 3 / 4)
	return w
}

func (uitu *uiThemeUtil) ShadowHeight(ff *fontutil.FontFace) int {
	lh := ff.LineHeightFloat()
	return int(lh * 2 / 5)
}

//----------
//----------
//----------

func cint(c int) color.RGBA {
	return imageutil.RgbaFromInt(c)
}

//----------

func newLinearInvertFn() func(color.Color) color.Color {
	//return imageutil.NewLinearInvertFn2(0.56, 2.5) // match gimp results
	return imageutil.NewLinearInvertFn2(0.80, 2.2) // darker, but better contrast then gimps
}
