package ui

import (
	"image/color"

	"github.com/jmigpin/editor/drawutil2/hsdrawer"
	"github.com/jmigpin/editor/imageutil"
)

var (
	White color.Color = color.RGBA{255, 255, 255, 255}
	Black color.Color = color.RGBA{0, 0, 0, 255}

	Red    color.Color = color.RGBA{255, 0, 0, 255}
	Yellow color.Color = color.RGBA{255, 153, 0, 255}
	Green  color.Color = color.RGBA{15, 173, 0, 255}
	Blue   color.Color = color.RGBA{0, 100, 181, 255}

	SquareColor            = ToolbarColors.Normal.Bg
	SquareActiveColor      = Black
	SquareExecutingColor   = Green
	SquareEditedColor      = color.RGBA{136, 136, 204, 255} // Blue
	SquareDiskChangesColor = Red
	SquareNotExistColor    = Yellow

	ScrollbarFgColor = color.Color(imageutil.Tint(Black, 0.70))
	ScrollbarBgColor = color.Color(imageutil.Tint(Black, 0.95))

	SeparatorColor         = Black
	RowInnerSeparatorColor = SquareEditedColor
)

var TextAreaColors = hsdrawer.Colors{
	Normal: hsdrawer.FgBg{Black, White},
	//Selection: hsdrawer.FgBg{nil, imageutil.Tint(Yellow, 0.50)},
	//Highlight: hsdrawer.FgBg{nil, imageutil.Tint(Blue, 0.70)},
	Selection: hsdrawer.FgBg{nil, color.RGBA{238, 238, 158, 255}},
	Highlight: hsdrawer.FgBg{nil, color.RGBA{198, 238, 158, 255}}, // analogous to selection bg
}

var ToolbarColors = hsdrawer.Colors{
	Normal:    hsdrawer.FgBg{Black, imageutil.Tint(Black, 0.95)},
	Selection: hsdrawer.FgBg{nil, TextAreaColors.Selection.Bg},
}

func AcmeColors() {
	TextAreaColors = hsdrawer.Colors{
		Normal:    hsdrawer.FgBg{Black, color.RGBA{255, 255, 234, 255}},
		Selection: hsdrawer.FgBg{nil, color.RGBA{238, 238, 158, 255}},
		Highlight: hsdrawer.FgBg{nil, color.RGBA{198, 238, 158, 255}}, // analogous to selection bg
	}
	ToolbarColors = hsdrawer.Colors{
		Normal:    hsdrawer.FgBg{Black, color.RGBA{234, 255, 255, 255}},
		Selection: hsdrawer.FgBg{nil, color.RGBA{158, 238, 238, 255}},
	}
	ScrollbarFgColor = color.RGBA{153, 153, 76, 255}
	ScrollbarBgColor = imageutil.Shade(TextAreaColors.Normal.Bg, 0.05)
	SquareColor = ToolbarColors.Normal.Bg
	SquareEditedColor = color.RGBA{136, 136, 204, 255}
	RowInnerSeparatorColor = SquareEditedColor
}
