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
	FlashDuration             = 500 * time.Millisecond
	ScrollBarLeft             = true
	ScrollBarWidth        int = 0 // 0=based on a portion of the font size
	SeparatorWidth            = 1
	TextAreaCommentsColor color.Color
)

var UITheme *uiTheme
var TTFontOptions truetype.Options

func init() {
	UITheme = NewUITheme()
	regularThemeFont()
	lightThemeColors()
}

type uiTheme struct {
	TextAreaTheme  widget.Theme
	ToolbarTheme   widget.Theme
	ScrollBarTheme widget.Theme
	NoRowColTheme  widget.Theme
	RowSquare      *RowSquareColors

	textAreaCommentsFg color.Color
}

func NewUITheme() *uiTheme {
	uit := &uiTheme{}
	uit.RowSquare = defaultRowSquareColors()
	return uit
}

func (uit *uiTheme) GetTextAreaCommentsFg() color.Color {
	if TextAreaCommentsColor != nil {
		return TextAreaCommentsColor
	}
	return uit.textAreaCommentsFg
}

// Used for: floatbox content
func (*uiTheme) HighlightSegmentColors() *widget.FgBg {
	// TODO: currently fixed for all themes
	fg := widget.Black
	bg := color.RGBA{158, 238, 238, 255} // light blue
	return &widget.FgBg{fg, bg}
}

// Used for:  row square color, textarea wrapline background.
func (*uiTheme) NoSelectionColors(t *widget.Theme) (_, _ color.Color) {
	pal := t.Palette()
	fg := pal.Normal.Fg
	bg := imageutil.TintOrShade(pal.Normal.Bg, 0.15)
	return fg, bg
}

func (uit *uiTheme) RowMinimumHeight(t *widget.Theme) int {
	return uit.FontFaceHeightInPixels(t.Font().Face(nil))
}
func (uit *uiTheme) RowSquareSize(t *widget.Theme) image.Point {
	lh := uit.FontFaceHeightInPixels(t.Font().Face(nil))
	w := int(float64(lh) * 3 / 4)
	return image.Point{w, lh}
}

func (uit *uiTheme) FontFaceHeightInPixels(face font.Face) int {
	m := face.Metrics()
	return (m.Ascent + m.Descent).Ceil()
}

func (uit *uiTheme) GetScrollBarWidth(t *widget.Theme) int {
	if ScrollBarWidth != 0 {
		return ScrollBarWidth
	}
	lh := uit.FontFaceHeightInPixels(t.Font().Face(nil))
	w := int(float64(lh) * 3 / 4)
	return w
}

func (uit *uiTheme) ShadowHeight() int {
	t := &UITheme.TextAreaTheme
	lh := uit.FontFaceHeightInPixels(t.Font().Face(nil))
	h := int(float64(lh) * 1 / 2)
	return h
}

func (uit *uiTheme) AnnotationsColors() (fg, bg color.Color) {
	pal := UITheme.TextAreaTheme.Palette()
	//return pal.Normal.Bg, pal.Normal.Fg

	hsv := imageutil.HSVModel.Convert(pal.Normal.Bg).(imageutil.HSV)
	d := uint8(float64(hsv.V) * 0.15)
	if hsv.V < 255/2 {
		hsv.V += d
	} else {
		hsv.V -= d
	}
	bg1 := color.RGBAModel.Convert(hsv).(color.RGBA)

	return pal.Normal.Fg, bg1
}

type RowSquareColors struct {
	Active             color.Color
	Executing          color.Color
	Edited             color.Color
	DiskChanges        color.Color
	NotExist           color.Color
	Duplicate          color.Color
	DuplicateHighlight color.Color
	Annotations        color.Color
	AnnotationsEdited  color.Color
}

func defaultRowSquareColors() *RowSquareColors {
	return &RowSquareColors{
		Active:             widget.Black,
		Executing:          color.RGBA{15, 173, 0, 255},        // dark green
		Edited:             color.RGBA{0, 0, 255, 255},         // blue
		DiskChanges:        color.RGBA{255, 0, 0, 255},         // red
		NotExist:           color.RGBA{255, 153, 0, 255},       // orange
		Duplicate:          color.RGBA{136, 136, 204, 255},     // blueish
		DuplicateHighlight: color.RGBA{255, 255, 0, 255},       // yellow
		Annotations:        color.RGBA{0xd3, 0x54, 0x00, 0xff}, // pumpkin
		AnnotationsEdited:  color.RGBA{255, 255, 0, 255},       // yellow
	}
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
		{"light", lightThemeColors},
		{"dark", darkThemeColors},
		{"acme", acmeThemeColors},
	},
}

func lightThemeColors() {
	textareaPal := &widget.Palette{
		Normal:    widget.FgBg{widget.Black, widget.White},
		Selection: widget.FgBg{nil, color.RGBA{238, 238, 158, 255}},
		Highlight: widget.FgBg{nil, color.RGBA{198, 238, 158, 255}},
	}
	toolbarPal := &widget.Palette{
		Normal:    widget.FgBg{widget.Black, color.RGBA{0xec, 0xf0, 0xf1, 0xff}}, // "clouds", prev was "grey 50"
		Selection: textareaPal.Selection,
	}

	UITheme.TextAreaTheme.SetPalette(textareaPal)
	UITheme.ToolbarTheme.SetPalette(toolbarPal)
	UITheme.NoRowColTheme.SetPalette(nil)
	UITheme.textAreaCommentsFg = color.RGBA{0x75, 0x75, 0x75, 0xff} // "grey 600"

	calcScrollBarTheme()
}

func calcScrollBarTheme() {
	//  colors based on normal.bg
	c1 := UITheme.TextAreaTheme.Palette().Normal.Bg
	var pal widget.Palette
	pal.Normal.Bg = imageutil.TintOrShade(c1, 0.05)
	pal.Normal.Fg = imageutil.TintOrShade(c1, 0.30)
	pal.Highlight.Fg = imageutil.TintOrShade(pal.Normal.Fg, 0.20)
	pal.Selection.Fg = imageutil.TintOrShade(pal.Normal.Fg, 0.40)
	UITheme.ScrollBarTheme.SetPalette(&pal)
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

	UITheme.TextAreaTheme.SetPalette(textareaPal)
	UITheme.ToolbarTheme.SetPalette(toolbarPal)
	UITheme.textAreaCommentsFg = color.RGBA{0xb8, 0xb8, 0xb8, 0xff}

	// no rows/cols theme
	var pal widget.Palette
	pal.Normal.Bg = imageutil.Shade(color.White, 0.30)
	UITheme.NoRowColTheme.SetPalette(&pal)

	calcScrollBarTheme()
}

func acmeThemeColors() {
	textareaPal := &widget.Palette{
		Normal:    widget.FgBg{widget.Black, color.RGBA{255, 255, 234, 255}},
		Selection: widget.FgBg{nil, color.RGBA{238, 238, 158, 255}},
		// bg is analogous to selection bg
		Highlight: widget.FgBg{nil, color.RGBA{198, 238, 158, 255}},
	}
	toolbarPal := &widget.Palette{
		Normal:    widget.FgBg{widget.Black, color.RGBA{234, 255, 255, 255}},
		Selection: textareaPal.Selection,
	}

	UITheme.TextAreaTheme.SetPalette(textareaPal)
	UITheme.ToolbarTheme.SetPalette(toolbarPal)
	UITheme.NoRowColTheme.SetPalette(nil)
	UITheme.textAreaCommentsFg = color.RGBA{0x75, 0x75, 0x75, 0xff} // "grey 600"

	// scrollbar
	{
		// TODO: use hsva to alter color and calc from the lighter color
		//pal.Normal.Fg = color.RGBA{193, 193, 147, 255}

		pal := *UITheme.TextAreaTheme.Palette()
		pal.Normal.Bg = imageutil.Shade(textareaPal.Normal.Bg, 0.05)
		darker := color.RGBA{153, 153, 76, 255}
		pal.Normal.Fg = imageutil.Tint(darker, 0.40)
		pal.Highlight.Fg = imageutil.Tint(darker, 0.20)
		pal.Selection.Fg = darker
		UITheme.ScrollBarTheme.SetPalette(&pal)
	}
}

var FontThemeCycler cycler = cycler{
	entries: []cycleEntry{
		{"regular", regularThemeFont},
		{"medium", mediumThemeFont},
		{"mono", monoThemeFont},
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
		&UITheme.TextAreaTheme,
		&UITheme.ToolbarTheme,
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
