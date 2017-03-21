package ui

import (
	"image/color"

	"github.com/jmigpin/editor/drawutil"
)

var (
	// these white/black take advantage of being RGBA
	White = color.RGBA{255, 255, 255, 255}
	Black = color.RGBA{0, 0, 0, 255}

	SeparatorColor = color.RGBA{0, 0, 0, 255}

	SquareColor          = color.RGBA{136, 136, 204, 255}
	SquareActiveColor    = Black
	SquareExecutingColor = color.RGBA{95, 204, 88, 255}
	SquareDirtyColor     = color.RGBA{204, 88, 92, 255}
	SquareColdColor      = color.RGBA{255, 255, 0, 255}
	SquareNotExistColor  = color.RGBA{204, 156, 88, 255}

	RowInnerSeparatorColor = SquareColor

	ScrollbarFgColor = color.RGBA{180, 180, 180, 255}
	ScrollbarBgColor = color.RGBA{241, 241, 241, 255}
)

var TextAreaColors = drawutil.Colors{
	Fg:          Black,
	Bg:          White,
	SelectionFg: nil,
	SelectionBg: color.RGBA{238, 238, 122, 255},
	HighlightFg: nil,
	HighlightBg: color.RGBA{209, 238, 162, 255},
	Comment:     color.RGBA{0, 100, 0, 255},
}

var ToolbarColors = drawutil.Colors{
	Fg:          Black,
	Bg:          color.RGBA{242, 242, 242, 255},
	SelectionBg: TextAreaColors.SelectionBg,
	HighlightBg: TextAreaColors.HighlightBg,
}
