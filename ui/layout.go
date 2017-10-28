package ui

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/uiutil/widget"
)

type Layout struct {
	widget.MultiLayer

	BgLayer   *widget.FlowLayout
	MenuLayer widget.Node

	Toolbar        *Toolbar
	MainMenuButton *MainMenuButton
	Cols           *Columns

	UI *UI
}

func (layout *Layout) Init(ui *UI) {
	layout.UI = ui

	layout.BgLayer = widget.NewFlowLayout()
	layout.BgLayer.SetWrapper(layout.BgLayer)

	mmb := NewMainMenuButton(ui)
	mmb.Label.Border.Left = 1
	mmb.Label.Border.Color = color.Black
	mmb.Label.Bg = ToolbarColors.Normal.Bg
	mmb.SetFill(false, true)
	layout.MainMenuButton = mmb

	layout.Toolbar = NewToolbar(ui, layout.BgLayer)
	layout.Toolbar.SetExpand(true, false)

	ttb := widget.NewFlowLayout()
	sep2 := widget.NewSpace(ui)
	sep2.SetFill(false, true)
	sep2.Size.X = 5
	sep2.Color = ToolbarColors.Normal.Bg
	ttb.Append(layout.Toolbar, sep2, mmb)

	sep := widget.NewSpace(ui)
	sep.SetExpand(true, false)
	sep.Size.Y = SeparatorWidth
	sep.Color = SeparatorColor

	layout.Cols = NewColumns(layout)
	layout.Cols.SetExpand(true, true)

	layout.BgLayer.YAxis = true
	layout.BgLayer.Append(ttb, sep, layout.Cols)

	// layer 1
	layout.MenuLayer = &mmb.FloatMenu

	// multi layer
	layout.Append(layout.BgLayer, layout.MenuLayer)
}

func (l *Layout) GoodColumnRowPlace() (*Column, *Row) {

	// TODO: accept optional row, or take into consideration active row
	// TODO: don't go too far away, stay close (active row)
	// TODO: belongs in Columns?

	var best struct {
		r       *image.Rectangle
		area    int
		col     *Column
		nextRow *Row
	}

	for _, c := range l.Cols.Columns() {
		rows := c.Rows()
		if len(rows) == 0 {
			s := c.Bounds().Size()
			a := s.X * s.Y
			if a > best.area {
				best.area = a
				best.col = c
				best.nextRow = nil
			}
		} else {
			for _, r := range rows {
				s := r.Bounds().Size()
				a := (s.X * s.Y)

				// endpercentlayout inserts rows and shares space with prev row, hence div by 2
				a2 := a / 2

				if a2 > best.area {
					best.area = a2
					best.col = c
					best.nextRow = nil
					r2, ok := r.NextRow()
					if ok {
						best.nextRow = r2
					}
				}
			}
		}
	}

	return best.col, best.nextRow
}
