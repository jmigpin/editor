package ui

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/widget"
)

type Layout struct {
	widget.MultiLayer
	UI              *UI
	Toolbar         *Toolbar
	MainMenuButton  *MainMenuButton
	ContextFloatBox *ContextFloatBox
	Cols            *Columns

	rowSepHandlesMark widget.Rectangle
	colSepHandlesMark widget.Rectangle
}

func (layout *Layout) Init(ui *UI) {
	layout.UI = ui
	layout.RootNodeWrapper(layout)

	//  background layer
	bgLayer := widget.NewBoxLayout()
	layout.Append(bgLayer)

	// column/row layer marks to be able to insert in a specific order
	layout.rowSepHandlesMark.SetHidden(true)
	layout.Append(&layout.rowSepHandlesMark)
	layout.colSepHandlesMark.SetHidden(true)
	layout.Append(&layout.colSepHandlesMark)

	// context floatbox layer
	layout.ContextFloatBox = NewContextFloatBox(layout)
	layout.Append(layout.ContextFloatBox)

	// floatmenu layer
	mmb := NewMainMenuButton(ui)
	layout.Append(mmb.FloatMenu)

	// setup background layer
	{
		bgLayer.YAxis = true

		// top toolbar
		ttb := widget.NewBoxLayout()
		bgLayer.Append(ttb)

		// toolbar
		layout.Toolbar = NewToolbar(ui, bgLayer)
		ttb.Append(layout.Toolbar)
		ttb.SetChildFlex(layout.Toolbar, true, false)

		// main menu button
		mmb.Label.Border.Left = 1
		mmb.Label.Border.Color = &SeparatorColor
		mmb.Label.Color = &ToolbarColors.Normal.Bg
		mmb.Label.Text.Color = &ToolbarColors.Normal.Fg
		ttb.Append(mmb)
		ttb.SetChildFill(mmb, false, true)
		layout.MainMenuButton = mmb

		// separator if there are no shadow
		if !ShadowsOn {
			sep := widget.NewRectangle(ui)
			sep.Size.Y = SeparatorWidth
			sep.Color = &SeparatorColor
			bgLayer.Append(sep)
		}

		layout.Cols = NewColumns(layout)
		bgLayer.Append(layout.Cols)
	}
}

func (l *Layout) InsertRowSepHandle(n widget.Node) {
	l.InsertBefore(n, &l.rowSepHandlesMark)
}
func (l *Layout) InsertColSepHandle(n widget.Node) {
	l.InsertBefore(n, &l.colSepHandlesMark)
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
			s := c.Bounds.Size()
			a := s.X * s.Y
			if a > best.area {
				best.area = a
				best.col = c
				best.nextRow = nil
			}
		} else {
			for _, r := range rows {
				s := r.Bounds.Size()
				a := (s.X * s.Y)

				// after insertion the space will be shared
				a2 := a / 2

				if a2 > best.area {
					best.area = a2
					best.col = c
					best.nextRow = r.NextRow()
				}
			}
		}
	}

	return best.col, best.nextRow
}
