package ui

import (
	"image"
	"image/color"

	"github.com/jmigpin/editor/uiutil/widget"
	"golang.org/x/image/colornames"
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
	layout.SetWrapper(layout)

	bgLayer := widget.NewFlowLayout()
	bgLayer.YAxis = true
	layout.Append(bgLayer)

	layout.colSepHandlesMark.SetHidden(true)
	layout.rowSepHandlesMark.SetHidden(true)
	layout.Append(&layout.colSepHandlesMark, &layout.rowSepHandlesMark)

	mmb := NewMainMenuButton(ui)
	mmb.Label.Border.Left = 1
	mmb.Label.Border.Color = &SeparatorColor
	mmb.Label.Color = &ToolbarColors.Normal.Bg
	mmb.Label.Text.Color = &ToolbarColors.Normal.Fg
	mmb.SetFill(false, true)
	layout.MainMenuButton = mmb

	layout.Toolbar = NewToolbar(ui, bgLayer)
	layout.Toolbar.SetExpand(true, false)

	ttb := widget.NewFlowLayout()
	ttb.Append(layout.Toolbar, mmb)

	layout.Cols = NewColumns(layout)
	layout.Cols.SetExpand(true, true)

	if ShadowsOn {
		bgLayer.Append(ttb, layout.Cols)
	} else {
		var sep widget.Rectangle
		sep.Init(ui)
		sep.SetExpand(true, false)
		sep.Size.Y = SeparatorWidth
		sep.Color = &SeparatorColor
		bgLayer.Append(ttb, &sep, layout.Cols)
	}

	layout.ContextFloatBox = NewContextFloatBox(layout)
	layout.ContextFloatBox.SetHidden(true)

	layout.Append(layout.ContextFloatBox, &mmb.FloatMenu)
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
	cfb.Label.Init(l.UI)
	cfb.FloatBox.Init(&cfb.Label)

	var c1 color.Color = colornames.Black
	var c2 color.Color = colornames.Orange

	cfb.Label.Text.Str = "testing"
	cfb.Label.Text.Color = &c1
	cfb.Label.Color = &c2
	cfb.Label.Pad.Set(5)
	cfb.Label.Pad.Color = &c2
	cfb.Label.Border.Set(1)
	cfb.Label.Border.Color = &c1

	return cfb
}
func (cfb *ContextFloatBox) SetStr(s string) {
	cfb.Label.Text.Str = s
	cfb.MarkNeedsPaint()
}
func (cfb *ContextFloatBox) CalcChildsBounds() {
	cfb.RefPoint = image.Point{40, 40}
	cfb.FloatBox.CalcChildsBounds()
}
