package ui

import (
	"image"

	"github.com/jmigpin/editor/uiutil/widget"
)

type Layout struct {
	widget.MultiLayer

	Toolbar         *Toolbar
	MainMenuButton  *MainMenuButton
	ContextFloatBox *ContextFloatBox
	Cols            *Columns

	UI *UI
}

func (layout *Layout) Init(ui *UI) {
	layout.UI = ui
	layout.SetWrapper(layout)

	bgLayer := widget.NewFlowLayout()

	mmb := NewMainMenuButton(ui)
	mmb.Label.Border.Left = 1
	mmb.Label.Border.Color = &SeparatorColor
	mmb.Label.Bg = &ToolbarColors.Normal.Bg
	mmb.Label.Text.Color = &ToolbarColors.Normal.Fg
	mmb.SetFill(false, true)
	layout.MainMenuButton = mmb

	layout.Toolbar = NewToolbar(ui, bgLayer)
	layout.Toolbar.SetExpand(true, false)

	ttb := widget.NewFlowLayout()
	var sep2 widget.Rectangle
	sep2.Init(ui)
	sep2.SetFill(false, true)
	sep2.Size.X = 5
	sep2.Color = &ToolbarColors.Normal.Bg
	ttb.Append(layout.Toolbar, &sep2, mmb)

	var sep widget.Rectangle
	sep.Init(ui)
	sep.SetExpand(true, false)
	sep.Size.Y = SeparatorWidth
	sep.Color = &SeparatorColor

	layout.Cols = NewColumns(layout)
	layout.Cols.SetExpand(true, true)

	bgLayer.YAxis = true
	bgLayer.Append(ttb, &sep, layout.Cols)

	// TODO: function that checks which elements of the lower layer need paint when an upper layer element needs paint

	layout.ContextFloatBox = NewContextFloatBox(layout)
	layout.ContextFloatBox.SetHidden(true)

	// multi layer
	layout.Append(bgLayer, layout.ContextFloatBox, &mmb.FloatMenu)
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

type ContextFloatBox struct {
	widget.FloatBox
	Label widget.Label
	l     *Layout
}

func NewContextFloatBox(l *Layout) *ContextFloatBox {
	cfb := &ContextFloatBox{l: l}
	//cfb.Label.Border.Color = &colornames.White

	cfb.FloatBox.Init(&l.MultiLayer, &cfb.Label)
	return cfb
}
func (cfb *ContextFloatBox) SetStr(s string) {
	cfb.Label.Text.Str = s
	//cfb.l.UpperLayerNeedsPaint(cfb)
	cfb.MarkNeedsPaint()
}
