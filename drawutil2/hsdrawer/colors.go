package hsdrawer

import (
	"image/color"

	"golang.org/x/image/colornames"
)

type Colors struct {
	Normal    FgBg
	Selection FgBg
	Highlight FgBg
	WrapLine  FgBg
}

type FgBg struct {
	Fg, Bg color.Color
}

var DefaultColors = Colors{
	Normal:    FgBg{color.Black, nil},
	Selection: FgBg{color.Black, colornames.Orange},
	Highlight: FgBg{color.Black, colornames.Aqua},
	WrapLine:  FgBg{color.White, colornames.Maroon},
}
