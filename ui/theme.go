package ui

import (
	"image"
	"image/color"
	"io/ioutil"
	"time"

	"github.com/golang/freetype/truetype"
	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/uiutil/widget"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

var (
	FlashDuration      = 500 * time.Millisecond
	ScrollBarLeft      = true
	ScrollBarWidth int = 0
	SeparatorWidth     = 1
)

var DefaultUITheme UITheme
var TTFontOptions truetype.Options

func init() {
	regularThemeFont()
	lightThemeColors()
	DefaultUITheme.RowSquare = defaultRowSquareTheme()
}

type UITheme struct {
	TextAreaTheme  widget.Theme
	ToolbarTheme   widget.Theme
	ScrollBarTheme widget.Theme
	NoRowColTheme  widget.Theme
	RowSquare      *RowSquareTheme
}

func (t *UITheme) HighlightSegment() *widget.FgBg {
	// TODO: currently fixed for all themes
	fg := widget.Black
	bg := color.RGBA{158, 238, 238, 255} // light blue
	return &widget.FgBg{fg, bg}
}

// Used for:  row square color, textarea wrapline background.
func NoSelectionColors(t *widget.Theme) *widget.FgBg {
	pal := t.Palette()
	fg := pal.Normal.Fg
	bg := imageutil.TintOrShade(pal.Normal.Bg, 0.15)
	return &widget.FgBg{fg, bg}
}

type RowSquareTheme struct {
	Active             color.Color
	Executing          color.Color
	Edited             color.Color
	DiskChanges        color.Color
	NotExist           color.Color
	Duplicate          color.Color
	HighlightDuplicate color.Color
}

func defaultRowSquareTheme() *RowSquareTheme {
	return &RowSquareTheme{
		Active:             widget.Black,
		Executing:          color.RGBA{15, 173, 0, 255},    // dark green
		Edited:             color.RGBA{0, 0, 255, 255},     // blue
		DiskChanges:        color.RGBA{255, 0, 0, 255},     // red
		NotExist:           color.RGBA{255, 153, 0, 255},   // orange
		Duplicate:          color.RGBA{136, 136, 204, 255}, // blueish
		HighlightDuplicate: color.RGBA{255, 255, 0, 255},   // yellow
	}
}

func RowMinimumHeight(t *widget.Theme) int {
	return FontFaceHeightInPixels(t.Font().Face(nil))
}
func RowSquareSize(t *widget.Theme) image.Point {
	lh := FontFaceHeightInPixels(t.Font().Face(nil))
	w := int(float64(lh) * 3 / 4)
	return image.Point{w, lh}
}

func FontFaceHeightInPixels(face font.Face) int {
	m := face.Metrics()
	return (m.Ascent + m.Descent).Ceil()
}

func GetScrollBarWidth(t *widget.Theme) int {
	if ScrollBarWidth != 0 {
		return ScrollBarWidth
	}
	lh := FontFaceHeightInPixels(t.Font().Face(nil))
	w := int(float64(lh) * 3 / 4)
	return w
}

func ShadowHeight() int {
	t := &DefaultUITheme.TextAreaTheme
	lh := FontFaceHeightInPixels(t.Font().Face(nil))
	h := int(float64(lh) * 1 / 2)
	return h
}

type cycler struct {
	index   string
	entries []cycleEntry
}

type cycleEntry struct {
	name string
	fn   func()
}

func (c *cycler) GetIndex(name string) (int, bool) {
	for i, e := range c.entries {
		if e.name == name {
			return i, true
		}
	}
	return -1, false
}
func (c *cycler) Cycle() {
	i := 0
	if c.index != "" {
		i, _ = c.GetIndex(c.index)
		i = (i + 1) % len(c.entries)
	}
	c.Set(c.entries[i].name)
}
func (c *cycler) Set(name string) {
	i, _ := c.GetIndex(name)
	c.entries[i].fn()
	c.index = name
}

var ColorThemeCycler cycler = cycler{
	entries: []cycleEntry{
		cycleEntry{"light", lightThemeColors},
		cycleEntry{"dark", darkThemeColors},
		cycleEntry{"acme", acmeThemeColors},
	},
}

func lightThemeColors() {
	textareaPal := &widget.Palette{
		Normal:    widget.FgBg{widget.Black, widget.White},
		Selection: widget.FgBg{widget.Black, color.RGBA{238, 238, 158, 255}},
		Highlight: widget.FgBg{widget.Black, color.RGBA{198, 238, 158, 255}},
	}
	toolbarPal := &widget.Palette{
		Normal:    widget.FgBg{widget.Black, color.RGBA{0xfa, 0xfa, 0xfa, 0xff}}, // "grey 50"
		Selection: textareaPal.Selection,
	}

	DefaultUITheme.TextAreaTheme.SetPalette(textareaPal)
	DefaultUITheme.ToolbarTheme.SetPalette(toolbarPal)
	DefaultUITheme.NoRowColTheme.SetPalette(nil)

	calcScrollBarTheme()
}

func calcScrollBarTheme() {
	//  colors based on normal.bg
	c1 := DefaultUITheme.TextAreaTheme.Palette().Normal.Bg
	var pal widget.Palette
	pal.Normal.Bg = imageutil.TintOrShade(c1, 0.05)
	pal.Normal.Fg = imageutil.TintOrShade(c1, 0.30)
	pal.Highlight.Fg = imageutil.TintOrShade(pal.Normal.Fg, 0.20)
	pal.Selection.Fg = imageutil.TintOrShade(pal.Normal.Fg, 0.40)
	DefaultUITheme.ScrollBarTheme.SetPalette(&pal)
}

func darkThemeColors() {
	textareaPal := &widget.Palette{
		Normal:    widget.FgBg{widget.White, widget.Black},
		Selection: widget.FgBg{widget.Black, color.RGBA{238, 238, 158, 255}},
		Highlight: widget.FgBg{widget.Black, color.RGBA{198, 238, 158, 255}},
	}
	toolbarPal := &widget.Palette{
		Normal:    widget.FgBg{widget.White, color.RGBA{0x80, 0x80, 0x80, 0xff}},
		Selection: textareaPal.Selection,
	}

	DefaultUITheme.TextAreaTheme.SetPalette(textareaPal)
	DefaultUITheme.ToolbarTheme.SetPalette(toolbarPal)

	// no rows/cols theme
	var pal widget.Palette
	pal.Normal.Bg = imageutil.Shade(color.White, 0.30)
	DefaultUITheme.NoRowColTheme.SetPalette(&pal)

	calcScrollBarTheme()
}

func acmeThemeColors() {
	textareaPal := &widget.Palette{
		Normal:    widget.FgBg{widget.Black, color.RGBA{255, 255, 234, 255}},
		Selection: widget.FgBg{widget.Black, color.RGBA{238, 238, 158, 255}},
		// bg is analogous to selection bg
		Highlight: widget.FgBg{widget.Black, color.RGBA{198, 238, 158, 255}},
	}
	toolbarPal := &widget.Palette{
		Normal:    widget.FgBg{widget.Black, color.RGBA{234, 255, 255, 255}},
		Selection: textareaPal.Selection,
	}

	DefaultUITheme.TextAreaTheme.SetPalette(textareaPal)
	DefaultUITheme.ToolbarTheme.SetPalette(toolbarPal)
	DefaultUITheme.NoRowColTheme.SetPalette(nil)

	// scrollbar
	{
		// TODO: use hsva to alter color and calc from the lighter color
		//pal.Normal.Fg = color.RGBA{193, 193, 147, 255}

		pal := *DefaultUITheme.TextAreaTheme.Palette()
		pal.Normal.Bg = imageutil.Shade(textareaPal.Normal.Bg, 0.05)
		darker := color.RGBA{153, 153, 76, 255}
		pal.Normal.Fg = imageutil.Tint(darker, 0.40)
		pal.Highlight.Fg = imageutil.Tint(darker, 0.20)
		pal.Selection.Fg = darker
		DefaultUITheme.ScrollBarTheme.SetPalette(&pal)
	}
}

var FontThemeCycler cycler = cycler{
	entries: []cycleEntry{
		cycleEntry{"regular", regularThemeFont},
		cycleEntry{"medium", mediumThemeFont},
		cycleEntry{"mono", monoThemeFont},
	},
}

func regularThemeFont() {
	loadThemeFont(goregular.TTF)
}
func mediumThemeFont() {
	loadThemeFont(gomedium.TTF)
}
func monoThemeFont() {
	loadThemeFont(gomono.TTF)
}

func loadThemeFont(b []byte) {
	themes := []*widget.Theme{
		&DefaultUITheme.TextAreaTheme,
		&DefaultUITheme.ToolbarTheme,
	}

	// clear previous fonts.
	for _, t := range themes {
		t.Font().Clear()
	}

	// load font
	tf := sureThemeFont(&TTFontOptions, b)
	for _, t := range themes {
		t.SetFont(tf)
	}
}

func sureThemeFont(opt *truetype.Options, b []byte) widget.ThemeFont {
	tf, err := widget.NewTTThemeFont(b, opt)
	if err != nil {
		panic(err)
	}
	return tf
}

func AddUserFont(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	userThemeFontBytes = b
	name := filename
	e := cycleEntry{name, userThemeFont}
	FontThemeCycler.entries = append(FontThemeCycler.entries, e)
	FontThemeCycler.Set(name)
	return nil
}

var userThemeFontBytes []byte

func userThemeFont() {
	loadThemeFont(userThemeFontBytes)
}
