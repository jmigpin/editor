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

//----------

var UITheme *uiTheme
var TTFontOptions truetype.Options

func init() {
	UITheme = NewUITheme()
	regularThemeFont()
	lightThemeColors()
}

//----------

type uiTheme struct {
	TextArea     widget.Theme
	Toolbar      widget.Theme
	Scrollbar    widget.Theme
	RowSquare    widget.Theme
	EmptySpaceBg widget.Theme
}

func NewUITheme() *uiTheme {
	uit := &uiTheme{}
	uit.RowSquare = *defaultRowSquareColors()
	return uit
}

func (uit *uiTheme) TextAreaCommentsFg() color.Color {
	if TextAreaCommentsColor != nil {
		return TextAreaCommentsColor
	}
	return uit.TextArea.Palette["comments_fg"]
}

//----------

var UIThemeUtil uiThemeUtil

type uiThemeUtil struct{}

func (uitu *uiThemeUtil) RowMinimumHeight(tf widget.ThemeFont) int {
	return uitu.FontFaceHeightInPixels(tf.Face(nil))
}
func (uitu *uiThemeUtil) RowSquareSize(tf widget.ThemeFont) image.Point {
	lh := uitu.FontFaceHeightInPixels(tf.Face(nil))
	w := int(float64(lh) * 3 / 4)
	return image.Point{w, lh}
}

func (uitu *uiThemeUtil) FontFaceHeightInPixels(face font.Face) int {
	m := face.Metrics()
	return (m.Ascent + m.Descent).Ceil()
}

func (uitu *uiThemeUtil) GetScrollBarWidth(tf widget.ThemeFont) int {
	if ScrollBarWidth != 0 {
		return ScrollBarWidth
	}
	lh := uitu.FontFaceHeightInPixels(tf.Face(nil))
	w := int(float64(lh) * 3 / 4)
	return w
}

func (uitu *uiThemeUtil) ShadowHeight() int {
	tf := widget.ThemeFontOrDefault(&UITheme.TextArea)
	lh := uitu.FontFaceHeightInPixels(tf.Face(nil))
	h := int(float64(lh) * 1 / 2)
	return h
}

//----------

func defaultRowSquareColors() *widget.Theme {
	pal := widget.Palette{
		"rs_active":              widget.Black,
		"rs_executing":           color.RGBA{15, 173, 0, 255},        // dark green
		"rs_edited":              color.RGBA{0, 0, 255, 255},         // blue
		"rs_disk_changes":        color.RGBA{255, 0, 0, 255},         // red
		"rs_not_exist":           color.RGBA{255, 153, 0, 255},       // orange
		"rs_duplicate":           color.RGBA{136, 136, 204, 255},     // blueish
		"rs_duplicate_highlight": color.RGBA{255, 255, 0, 255},       // yellow
		"rs_annotations":         color.RGBA{0xd3, 0x54, 0x00, 0xff}, // pumpkin
		"rs_annotations_edited":  color.RGBA{255, 255, 0, 255},       // yellow
	}
	return &widget.Theme{Palette: pal}
}

//----------

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

//----------

var ColorThemeCycler cycler = cycler{
	entries: []cycleEntry{
		{"light", lightThemeColors},
		{"dark", darkThemeColors},
		{"acme", acmeThemeColors},
	},
}

//----------

func lightThemeColors() {
	textareaPal := widget.Palette{
		"fg":           widget.Black,
		"bg":           widget.White,
		"selection_fg": nil,
		"selection_bg": color.RGBA{238, 238, 158, 255},
		"highlight_fg": nil,
		"highlight_bg": color.RGBA{198, 238, 158, 255},
		"comments_fg":  color.RGBA{0x75, 0x75, 0x75, 0xff}, // "grey 600"

		// used in: square background, wrapline runes.
		"noselection_fg": widget.Black,
		"noselection_bg": imageutil.TintOrShade(widget.White, 0.15),

		"annotations_fg": nil,
		"annotations_bg": imageutil.TintOrShade(widget.White, 0.15),
		"segments_fg":    widget.Black,
		"segments_bg":    color.RGBA{158, 238, 238, 255}, // light blue
	}
	UITheme.TextArea.Palette = textareaPal

	tbBg := color.RGBA{0xec, 0xf0, 0xf1, 0xff} // "clouds" grey
	toolbarPal := widget.Palette{
		"fg":             widget.Black,
		"bg":             tbBg,
		"selection_fg":   textareaPal["selection_fg"],
		"selection_bg":   textareaPal["selection_bg"],
		"noselection_fg": widget.Black,
		"noselection_bg": imageutil.TintOrShade(tbBg, 0.15),
	}
	UITheme.Toolbar.Palette = toolbarPal

	calcScrollBarTheme()

	UITheme.EmptySpaceBg.Palette = widget.Palette{
		"bg": widget.White,
	}
}

func calcScrollBarTheme() {
	c1 := UITheme.TextArea.Palette["bg"] // based on bg

	pal := widget.MakePalette()
	pal["bg"] = imageutil.TintOrShade(c1, 0.05)
	pal["normal"] = imageutil.TintOrShade(c1, 0.30)
	pal["select"] = imageutil.TintOrShade(pal["normal"], 0.40)
	pal["highlight"] = imageutil.TintOrShade(pal["normal"], 0.20)
	UITheme.Scrollbar.Palette = pal
}

//----------

func darkThemeColors() {
	textareaPal := widget.Palette{
		"fg":             widget.White,
		"bg":             widget.Black,
		"selection_fg":   widget.Black,
		"selection_bg":   color.RGBA{238, 238, 158, 255},
		"highlight_fg":   widget.Black,
		"highlight_bg":   color.RGBA{198, 238, 158, 255},
		"comments_fg":    color.RGBA{0xb8, 0xb8, 0xb8, 0xff},
		"noselection_fg": nil,
		"noselection_bg": imageutil.TintOrShade(widget.Black, 0.35),
		"annotations_fg": nil,
		"annotations_bg": imageutil.TintOrShade(widget.Black, 0.35),
		"segments_fg":    widget.Black,
		"segments_bg":    color.RGBA{158, 238, 238, 255}, // light blue
	}
	UITheme.TextArea.Palette = textareaPal

	toolbarPal := widget.Palette{
		"fg":           widget.White,
		"bg":           color.RGBA{0x80, 0x80, 0x80, 0xff},
		"selection_fg": textareaPal["selection_fg"],
		"selection_bg": textareaPal["selection_bg"],
	}
	toolbarPal["noselection_bg"] = imageutil.TintOrShade(toolbarPal["bg"], 0.35)
	UITheme.Toolbar.Palette = toolbarPal

	calcScrollBarTheme()

	UITheme.EmptySpaceBg.Palette = widget.Palette{
		"bg": imageutil.Shade(color.White, 0.30),
	}
}

//----------

func acmeThemeColors() {
	taBg := color.RGBA{255, 255, 234, 255}
	textareaPal := widget.Palette{
		"fg":             widget.Black,
		"bg":             taBg,
		"selection_fg":   nil,
		"selection_bg":   color.RGBA{238, 238, 158, 255},
		"highlight_fg":   nil,
		"highlight_bg":   color.RGBA{198, 238, 158, 255},     // analogous to selection bg
		"comments_fg":    color.RGBA{0x75, 0x75, 0x75, 0xff}, // "grey 600"
		"noselection_fg": widget.Black,
		"noselection_bg": imageutil.TintOrShade(taBg, 0.15),
		"annotations_fg": widget.Black,
		"annotations_bg": imageutil.TintOrShade(taBg, 0.15),
		"segments_fg":    widget.Black,
		"segments_bg":    color.RGBA{158, 238, 238, 255}, // light blue
	}
	UITheme.TextArea.Palette = textareaPal

	tbBg := color.RGBA{234, 255, 255, 255}
	toolbarPal := widget.Palette{
		"fg":             widget.Black,
		"bg":             tbBg,
		"selection_fg":   textareaPal["selection_fg"],
		"selection_bg":   textareaPal["selection_bg"],
		"noselection_fg": widget.Black,
		"noselection_bg": imageutil.TintOrShade(tbBg, 0.15),
	}
	UITheme.Toolbar.Palette = toolbarPal

	// scrollbar
	{
		pal := widget.MakePalette()
		pal["bg"] = imageutil.Shade(textareaPal["bg"], 0.05)

		darker := color.RGBA{153, 153, 76, 255}
		pal["normal"] = imageutil.Tint(darker, 0.40)
		pal["highlight"] = imageutil.Tint(darker, 0.20)
		pal["select"] = darker

		UITheme.Scrollbar.Palette = pal
	}

	UITheme.EmptySpaceBg.Palette = widget.Palette{
		"bg": color.White,
	}
}

//----------

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
		&UITheme.TextArea,
		&UITheme.Toolbar,
		&UITheme.EmptySpaceBg, // helps calculate square size
	}

	// clear previous fonts.
	for _, t := range themes {
		if t.Font != nil {
			t.Font.Clear()
		}
	}

	// load font
	tf := sureThemeFont(&TTFontOptions, b)
	for _, t := range themes {
		t.Font = tf
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
