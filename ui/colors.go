package ui

import (
	"image/color"

	"github.com/jmigpin/editor/drawutil"
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

	RowInnerSeparatorColor = tint(&Blue, 0.50)

	SquareColor            = ToolbarColors.Bg
	SquareActiveColor      = Black
	SquareExecutingColor   = Green
	SquareEditedColor      = Blue
	SquareDiskChangesColor = Red
	SquareNotExistColor    = Yellow

	ScrollbarFgColor = tint(&Black, 0.70)
	ScrollbarBgColor = tint(&Black, 0.95)
)

var TextAreaColors = drawutil.Colors{
	Fg:          Black,
	Bg:          White,
	SelectionBg: tint(&Yellow, 0.50),
	HighlightBg: tint(&Blue, 0.70),
}

var ToolbarColors = drawutil.Colors{
	Fg:          Black,
	Bg:          tint(&Black, 0.95),
	SelectionBg: TextAreaColors.SelectionBg,
}

func tint(c0 *color.RGBA, v float64) color.RGBA {
	c := *c0
	c.R += uint8(v * float64((255 - c.R)))
	c.G += uint8(v * float64((255 - c.G)))
	c.B += uint8(v * float64((255 - c.B)))
	return c
}
