package ui

import (
	"image/color"

	"github.com/jmigpin/editor/drawutil"
	"github.com/jmigpin/editor/imageutil"
)

var (
	// these white/black take advantage of being RGBA
	White = color.RGBA{255, 255, 255, 255}
	Black = color.RGBA{0, 0, 0, 255}

	// tetradic color scheme
	// http://www.tigercolor.com/color-lab/color-theory/color-theory-intro.htm
	Red    = color.RGBA{255, 0, 0, 255}
	Yellow = color.RGBA{255, 153, 0, 255}
	Green  = color.RGBA{15, 173, 0, 255}
	Blue   = color.RGBA{0, 100, 181, 255}

	SeparatorColor = Black

	RowInnerSeparatorColor = imageutil.Tint(&Blue, 0.50)

	SquareColor            = ToolbarColors.Bg
	SquareActiveColor      = Black
	SquareExecutingColor   = Green
	SquareEditedColor      = Blue
	SquareDiskChangesColor = Red
	SquareNotExistColor    = Yellow

	ScrollbarFgColor = imageutil.Tint(&Black, 0.70)
	ScrollbarBgColor = imageutil.Tint(&Black, 0.95)
)

var TextAreaColors = drawutil.Colors{
	Fg:          Black,
	Bg:          White,
	SelectionBg: imageutil.Tint(&Yellow, 0.50),
	HighlightBg: imageutil.Tint(&Blue, 0.70),
}

var ToolbarColors = drawutil.Colors{
	Fg:          Black,
	Bg:          imageutil.Tint(&Black, 0.95),
	SelectionBg: TextAreaColors.SelectionBg,
}
