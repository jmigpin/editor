package ui

import (
	"image/color"

	"github.com/jmigpin/editor/drawutil2/hsdrawer"
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

	SquareColor            = ToolbarColors.Normal.Bg
	SquareActiveColor      = Black
	SquareExecutingColor   = Green
	SquareEditedColor      = Blue
	SquareDiskChangesColor = Red
	SquareNotExistColor    = Yellow

	ScrollbarFgColor = color.Color(imageutil.Tint(&Black, 0.70))
	ScrollbarBgColor = color.Color(imageutil.Tint(&Black, 0.95))
)

var TextAreaColors = hsdrawer.Colors{
	Normal:    hsdrawer.FgBg{Black, White},
	Selection: hsdrawer.FgBg{nil, imageutil.Tint(&Yellow, 0.50)},
	Highlight: hsdrawer.FgBg{nil, imageutil.Tint(&Blue, 0.70)},
}

var ToolbarColors = hsdrawer.Colors{
	Normal:    hsdrawer.FgBg{Black, imageutil.Tint(&Black, 0.95)},
	Selection: hsdrawer.FgBg{nil, TextAreaColors.Selection.Bg},
}

func AcmeColors() {
	TextAreaColors = hsdrawer.Colors{
		Normal:    hsdrawer.FgBg{Black, color.RGBA{255, 255, 234, 255}},
		Selection: hsdrawer.FgBg{nil, imageutil.Tint(&Yellow, 0.50)},
		Highlight: hsdrawer.FgBg{nil, imageutil.Tint(&Blue, 0.70)},
	}
	ToolbarColors = hsdrawer.Colors{
		Normal:    hsdrawer.FgBg{Black, color.RGBA{234, 255, 255, 255}},
		Selection: hsdrawer.FgBg{nil, TextAreaColors.Selection.Bg},
	}
	SquareColor = ToolbarColors.Normal.Bg
	ScrollbarFgColor = color.RGBA{153, 153, 76, 255}
	ScrollbarBgColor = TextAreaColors.Normal.Bg
}
