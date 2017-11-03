package ui

import (
	"image/color"

	"github.com/jmigpin/editor/drawutil2/hsdrawer"
	"github.com/jmigpin/editor/imageutil"
	"golang.org/x/image/colornames"
)

// This is temporary until a theme structure is in place.

var (
	White  color.Color = color.RGBA{255, 255, 255, 255}
	Black  color.Color = color.RGBA{0, 0, 0, 255}
	Red    color.Color = color.RGBA{255, 0, 0, 255}
	Yellow color.Color = color.RGBA{255, 153, 0, 255}
	Green  color.Color = color.RGBA{15, 173, 0, 255}
	Blue   color.Color = color.RGBA{0, 100, 181, 255}
)

var TextAreaColors hsdrawer.Colors
var ToolbarColors hsdrawer.Colors
var SquareColor color.Color
var SquareActiveColor color.Color
var SquareExecutingColor color.Color
var SquareEditedColor color.Color
var SquareDiskChangesColor color.Color
var SquareNotExistColor color.Color
var ScrollbarFgColor color.Color
var ScrollbarBgColor color.Color
var SeparatorColor color.Color
var RowInnerSeparatorColor color.Color
var ColumnBgColor color.Color

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
		Selection: hsdrawer.FgBg{nil, color.RGBA{238, 238, 158, 255}},
		Highlight: hsdrawer.FgBg{nil, color.RGBA{198, 238, 158, 255}}, // analogous to selection bg
		WrapLine:  hsdrawer.FgBg{nil, color.RGBA{200, 200, 200, 255}}, // between 198,245
	}
	ToolbarColors = hsdrawer.Colors{
		Normal:    hsdrawer.FgBg{Black, color.RGBA{234, 255, 255, 255}},
		Selection: hsdrawer.FgBg{nil, color.RGBA{158, 238, 238, 255}},
		WrapLine:  TextAreaColors.WrapLine,
	}

	SquareColor = ToolbarColors.Normal.Bg
	SquareActiveColor = Black
	SquareExecutingColor = Green
	SquareEditedColor = color.RGBA{136, 136, 204, 255} // Blue
	SquareDiskChangesColor = Red
	SquareNotExistColor = Yellow

	ScrollbarFgColor = color.RGBA{153, 153, 76, 255}
	ScrollbarBgColor = imageutil.Shade(TextAreaColors.Normal.Bg, 0.05)

	SeparatorColor = Black
	RowInnerSeparatorColor = SquareEditedColor
	ColumnBgColor = White
}

func LightThemeColors() {
	defer func() { colorThemeIndex = 1 }()
	AcmeThemeColors()
	TextAreaColors = hsdrawer.Colors{
		Normal:    hsdrawer.FgBg{Black, White},
		Selection: hsdrawer.FgBg{nil, color.RGBA{238, 238, 158, 255}},
		Highlight: hsdrawer.FgBg{nil, color.RGBA{198, 238, 158, 255}},
		WrapLine:  hsdrawer.FgBg{nil, color.RGBA{200, 200, 200, 255}},
	}
	ToolbarColors = hsdrawer.Colors{
		Normal:    hsdrawer.FgBg{Black, imageutil.Tint(Black, 0.95)},
		Selection: hsdrawer.FgBg{nil, TextAreaColors.Selection.Bg},
		WrapLine:  TextAreaColors.WrapLine,
	}

	SquareColor = ToolbarColors.Normal.Bg

	ScrollbarFgColor = color.Color(imageutil.Tint(Black, 0.70))
	ScrollbarBgColor = imageutil.Shade(TextAreaColors.Normal.Bg, 0.05)
}

func DarkThemeColors() {
	defer func() { colorThemeIndex = 2 }()
	LightThemeColors()
	TextAreaColors = hsdrawer.Colors{
		Normal:    hsdrawer.FgBg{White, colornames.Grey},
		Selection: hsdrawer.FgBg{Black, color.RGBA{238, 238, 158, 255}},
		Highlight: hsdrawer.FgBg{Black, color.RGBA{198, 238, 158, 255}},
		WrapLine:  hsdrawer.FgBg{Black, color.RGBA{200, 200, 200, 255}},
	}
	ToolbarColors = hsdrawer.Colors{
		Normal:    hsdrawer.FgBg{White, Black},
		Selection: hsdrawer.FgBg{Black, TextAreaColors.Selection.Bg},
		WrapLine:  TextAreaColors.WrapLine,
	}

	SquareColor = ToolbarColors.Normal.Bg
	SquareActiveColor = White

	ScrollbarFgColor = color.Color(imageutil.Tint(Black, 0.70))
	ScrollbarBgColor = imageutil.Shade(TextAreaColors.Normal.Bg, 0.05)

	SeparatorColor = White
	RowInnerSeparatorColor = SquareEditedColor
	ColumnBgColor = Black
}
