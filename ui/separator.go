package ui

import (
	"image/color"

	"github.com/jmigpin/editor/uiutil"
)

type Separator struct {
	C     uiutil.Container
	ui    *UI
	color color.Color
}

func NewSeparator(ui *UI, size int, c color.Color) *Separator {
	s := &Separator{ui: ui, color: c}
	s.C.PaintFunc = s.paint
	s.C.Style.MainSize = &size
	return s
}
func (s *Separator) paint() {
	s.ui.FillRectangle(&s.C.Bounds, s.color)
}
