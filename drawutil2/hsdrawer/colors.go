package hsdrawer

import (
	"image/color"

	"golang.org/x/image/colornames"
)

type Colors struct {
	Normal    FgBg
	Selection FgBg
	Highlight FgBg
}

type FgBg struct {
	Fg, Bg color.Color
}

func DefaultColors() Colors {
	return Colors{
		Normal:    FgBg{color.Black, nil},
		Selection: FgBg{color.Black, colornames.Orange},
		Highlight: FgBg{color.Black, colornames.Aqua},
	}
}
