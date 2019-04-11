package ui

import (
	"fmt"
	"image"
	"image/color"
	"io/ioutil"

	"github.com/golang/freetype/truetype"
	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/uiutil/widget"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

var (
	ScrollBarLeft             = true
	ScrollBarWidth        int = 0 // 0=based on a portion of the font size
	TextAreaCommentsColor color.Color
	TextAreaStringsColor  color.Color
)

const (
	separatorWidth = 1 // col/row separators width
)

//----------

// Palette with user supplied color options that should override themes.
func userPalette() widget.Palette {
	pal := widget.Palette{}
	if TextAreaCommentsColor != nil {
		pal["text_colorize_comments_fg"] = TextAreaCommentsColor
	}
	if TextAreaStringsColor != nil {
		pal["text_colorize_string_fg"] = TextAreaStringsColor
	}
	return pal
}

//----------

func lightThemeColors(node widget.Node) {
	pal := widget.Palette{
		"text_cursor_fg":            cint(0x0),
		"text_fg":                   cint(0x0),
		"text_bg":                   cint(0xffffff),
		"text_selection_fg":         nil,
		"text_selection_bg":         cint(0xeeee9e), // yellow
		"text_colorize_string_fg":   nil,
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
	node.Embed().SetThemePalette(pal)
}

//----------

func darkThemeColors(node widget.Node) {
	pal := widget.Palette{
		"text_cursor_fg":            cint(0xffffff),
		"text_fg":                   cint(0xffffff),
		"text_bg":                   cint(0x0),
		"text_selection_fg":         cint(0xffffff),
		"text_selection_bg":         cint(0xafa753), // yellow
		"text_colorize_string_fg":   nil,
		"text_colorize_comments_fg": cint(0xb8b8b8),
		"text_highlightword_bg":     cint(0x58842d), // green
		"text_wrapline_fg":          cint(0xffffff),
		"text_wrapline_bg":          cint(0x595959),

		"toolbar_text_fg":          cint(0xffffff),
		"toolbar_text_bg":          cint(0x808080),
		"toolbar_text_wrapline_bg": imageutil.Shade(cint(0x808080), 0.20),

		"scrollbar_bg":        imageutil.Tint(cint(0x0), 0.20),
		"scrollhandle_normal": imageutil.Tint(cint(0x0), 0.40),
		"scrollhandle_hover":  imageutil.Tint(cint(0x0), 0.50),
		"scrollhandle_select": imageutil.Tint(cint(0x0), 0.60),

		"column_norows_rect":  imageutil.Tint(cint(0x0), 0.10),
		"columns_nocols_rect": imageutil.Tint(cint(0x0), 0.10),
		"colseparator_rect":   cint(0x0),
		"rowseparator_rect":   cint(0x0),
		"shadowsep_rect":      cint(0x0),

		"columnsquare": imageutil.Shade(cint(0x808080), 0.20),
		"rowsquare":    imageutil.Shade(cint(0x808080), 0.20),

		"mm_text_bg":          cint(0x808080),
		"mm_button_hover_bg":  imageutil.Tint(cint(0x808080), 0.10),
		"mm_button_down_bg":   imageutil.Tint(cint(0x808080), 0.20),
		"mm_button_sticky_bg": imageutil.Tint(cint(0x808080), 0.40),
		"mm_border":           cint(0x0),
		"mm_content_pad":      cint(0x808080),
		"mm_content_border":   cint(0x0),

		"contextfloatbox_border": cint(0xffffff),
	}

	pal.Merge(rowSquarePalette())
	pal.Merge(userPalette())
	node.Embed().SetThemePalette(pal)
}

//----------

func acmeThemeColors(node widget.Node) {
	pal := widget.Palette{
		"text_cursor_fg":            cint(0x0),
		"text_fg":                   cint(0x0),
		"text_bg":                   cint(0xffffea),
		"text_selection_fg":         nil,
		"text_selection_bg":         cint(0xeeee9e), // yellow
		"text_colorize_string_fg":   nil,
		"text_colorize_comments_fg": cint(0x757575), // grey 600
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

		"column_norows_rect":  cint(0xffffff),
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
	node.Embed().SetThemePalette(pal)
}

//----------

func rowSquarePalette() widget.Palette {
	pal := widget.Palette{
		"rs_active":              cint(0x0),
		"rs_executing":           color.RGBA{15, 173, 0, 255},        // dark green
		"rs_edited":              color.RGBA{0, 0, 255, 255},         // blue
		"rs_disk_changes":        color.RGBA{255, 0, 0, 255},         // red
		"rs_not_exist":           color.RGBA{255, 153, 0, 255},       // orange
		"rs_duplicate":           color.RGBA{136, 136, 204, 255},     // blueish
		"rs_duplicate_highlight": color.RGBA{255, 255, 0, 255},       // yellow
		"rs_annotations":         color.RGBA{0xd3, 0x54, 0x00, 0xff}, // pumpkin
		"rs_annotations_edited":  color.RGBA{255, 255, 0, 255},       // yellow
	}
	return pal
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
	_, err := ThemeFont(filename)
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
	// close previous font faces
	tf0 := node.Embed().TreeThemeFont()
	tf0.CloseFaces()

	tf, err := ThemeFont(name)
	if err != nil {
		return err
	}
	node.Embed().SetThemeFont(tf)
	return nil
}

//----------

var TTFontOptions truetype.Options

func ThemeFont(name string) (widget.ThemeFont, error) {
	fb, err := fontBytes(name)
	if err != nil {
		return nil, err
	}
	return widget.NewTTThemeFont(fb, &TTFontOptions)
}

//----------

var fontBytesCache = map[string][]byte{}

func fontBytes(name string) ([]byte, error) {
	switch name {
	case "regular":
		return goregular.TTF, nil
	case "medium":
		return gomedium.TTF, nil
	case "mono":
		return gomono.TTF, nil
	default:
		// TODO: should clear if last instance is closed
		if b, ok := fontBytesCache[name]; ok {
			return b, nil
		}
		b, err := ioutil.ReadFile(name)
		if err != nil {
			return nil, err
		}
		fontBytesCache[name] = b
		return b, nil
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

type cycleEntry struct {
	name string
	fn   func(widget.Node)
}

//----------

var UIThemeUtil uiThemeUtil

type uiThemeUtil struct{}

func (uitu *uiThemeUtil) RowMinimumHeight(tf widget.ThemeFont) int {
	return uitu.LineHeight(tf.Face(nil))
}
func (uitu *uiThemeUtil) RowSquareSize(tf widget.ThemeFont) image.Point {
	lh := uitu.LineHeight(tf.Face(nil))
	w := int(float64(lh) * 3 / 4)
	return image.Point{w, lh}
}

func (uitu *uiThemeUtil) LineHeight(face font.Face) int {
	m := face.Metrics()
	return drawutil.LineHeightInt(&m)
}

func (uitu *uiThemeUtil) GetScrollBarWidth(tf widget.ThemeFont) int {
	if ScrollBarWidth != 0 {
		return ScrollBarWidth
	}
	lh := uitu.LineHeight(tf.Face(nil))
	w := int(float64(lh) * 3 / 4)
	return w
}

func (uitu *uiThemeUtil) ShadowHeight(tf widget.ThemeFont) int {
	lh := uitu.LineHeight(tf.Face(nil))
	h := int(float64(lh) * 2 / 5)
	return h
}

//----------

func cint(c int) color.RGBA {
	return imageutil.IntRGBA(c)
}
