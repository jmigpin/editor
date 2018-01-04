package ui

import (
	"image/color"

	"github.com/jmigpin/editor/drawutil/hsdrawer"
	"github.com/jmigpin/editor/imageutil"
	"golang.org/x/image/colornames"
)

// This is temporary until a theme structure is in place.

var (
	White color.Color = color.RGBA{255, 255, 255, 255}
	Black color.Color = color.RGBA{0, 0, 0, 255}
)

var TextAreaColors hsdrawer.Colors
var ToolbarColors hsdrawer.Colors
var SquareColor color.Color
var SquareActiveColor color.Color
var SquareExecutingColor color.Color
var SquareEditedColor color.Color
var SquareDiskChangesColor color.Color
var SquareNotExistColor color.Color
var SquareDuplicateColor color.Color
var SquareHighlightDuplicateColor color.Color
var ScrollbarFgColor color.Color
var ScrollbarBgColor color.Color
var SeparatorColor color.Color
var RowInnerSeparatorColor color.Color
var ColumnBgColor color.Color
var HighlightSegmentBgColor color.Color

var colorThemeIndex = 0
var colorThemeFn = []func(){AcmeThemeColors, LightThemeColors, DarkThemeColors}

func CycleColorTheme() {
	colorThemeIndex = (colorThemeIndex + 1) % len(colorThemeFn)
	colorThemeFn[colorThemeIndex]()
}

func AcmeThemeColors() {
	defer func() { colorThemeIndex = 0 }()

	TextAreaColors = hsdrawer.Colors{
		Normal:    hsdrawer.FgBg{Black, color.RGBA{255, 255, 234, 255}},
		Selection: hsdrawer.FgBg{Black, color.RGBA{238, 238, 158, 255}},
		Highlight: hsdrawer.FgBg{Black, color.RGBA{198, 238, 158, 255}}, // analogous to selection bg
	}
	ToolbarColors = hsdrawer.Colors{
		Normal: hsdrawer.FgBg{Black, color.RGBA{234, 255, 255, 255}},
		//Selection: hsdrawer.FgBg{Black, color.RGBA{158, 238, 238, 255}},
		Selection: TextAreaColors.Selection,
	}

	calcOtherColors()

	SquareActiveColor = Black
	SquareExecutingColor = color.RGBA{15, 173, 0, 255}           // dark green
	SquareEditedColor = color.RGBA{0, 0, 255, 255}               // blue
	SquareDiskChangesColor = color.RGBA{255, 0, 0, 255}          // red
	SquareNotExistColor = color.RGBA{255, 153, 0, 255}           // orange
	SquareDuplicateColor = color.RGBA{136, 136, 204, 255}        // blueish
	SquareHighlightDuplicateColor = color.RGBA{255, 255, 0, 255} // yellow

	ScrollbarFgColor = color.RGBA{153, 153, 76, 255}
	SeparatorColor = Black
	RowInnerSeparatorColor = color.RGBA{136, 136, 204, 255} // blueish
	ColumnBgColor = White
	HighlightSegmentBgColor = color.RGBA{158, 238, 238, 255} // light blue
}

func LightThemeColors() {
	defer func() { colorThemeIndex = 1 }()

	AcmeThemeColors()

	TextAreaColors = hsdrawer.Colors{
		Normal:    hsdrawer.FgBg{Black, White},
		Selection: hsdrawer.FgBg{Black, color.RGBA{238, 238, 158, 255}},
		Highlight: hsdrawer.FgBg{Black, color.RGBA{198, 238, 158, 255}},
	}
	ToolbarColors = hsdrawer.Colors{
		Normal:    hsdrawer.FgBg{Black, imageutil.Tint(Black, 0.95)},
		Selection: hsdrawer.FgBg{Black, TextAreaColors.Selection.Bg},
	}

	calcOtherColors()

	ScrollbarFgColor = color.Color(imageutil.Tint(Black, 0.70))
	RowInnerSeparatorColor = Black
}

func DarkThemeColors() {
	defer func() { colorThemeIndex = 2 }()

	LightThemeColors()

	TextAreaColors = hsdrawer.Colors{
		Normal:    hsdrawer.FgBg{White, Black},
		Selection: hsdrawer.FgBg{Black, color.RGBA{238, 238, 158, 255}},
		Highlight: hsdrawer.FgBg{Black, color.RGBA{198, 238, 158, 255}},
	}
	ToolbarColors = hsdrawer.Colors{
		Normal:    hsdrawer.FgBg{White, colornames.Grey},
		Selection: hsdrawer.FgBg{Black, TextAreaColors.Selection.Bg},
	}

	calcOtherColors()

	SquareActiveColor = White
	ScrollbarFgColor = color.Color(imageutil.Tint(Black, 0.70))
	SeparatorColor = White
	RowInnerSeparatorColor = White
	ColumnBgColor = colornames.Grey
}

func calcOtherColors() {
	setWrapLineColors(&TextAreaColors)
	setWrapLineColors(&ToolbarColors)
	SquareColor = imageutil.TintOrShade(ToolbarColors.Normal.Bg, 0.15)
	ScrollbarBgColor = imageutil.TintOrShade(TextAreaColors.Normal.Bg, 0.05)
}

func setWrapLineColors(c *hsdrawer.Colors) {
	c.WrapLine = hsdrawer.FgBg{
		c.Normal.Fg,
		imageutil.TintOrShade(c.Normal.Bg, 0.15),
	}
}
