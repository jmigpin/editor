package ui

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/widget"
)

// User Interface root (top) node.
type Root struct {
	*widget.MultiLayer
	UI              *UI
	Toolbar         *Toolbar
	MainMenuButton  *MainMenuButton
	ContextFloatBox *ContextFloatBox
	Cols            *Columns
}

func NewRoot(ui *UI) *Root {
	return &Root{MultiLayer: widget.NewMultiLayer(), UI: ui}
}

func (root *Root) Init() {
	//  background layer
	bgLayer := widget.NewBoxLayout()
	bgLayer.YAxis = true
	root.BgLayer.Append(bgLayer)

	// context floatbox layer
	//root.ContextFloatBox = NewContextFloatBox(root)
	//root.FloatLayer.Append(root.ContextFloatBox)

	// background layer
	{
		// top toolbar
		{
			ttb := widget.NewBoxLayout()
			bgLayer.Append(ttb)

			// toolbar
			root.Toolbar = NewToolbar(root.UI)
			ttb.Append(root.Toolbar)
			ttb.SetChildFlex(root.Toolbar, true, false)

			// main menu button
			mmb := NewMainMenuButton(root)
			mmb.Label.Border.Left = 1
			ttb.Append(mmb)
			ttb.SetChildFill(mmb, false, true)
			root.MainMenuButton = mmb
		}

		//// separator if there are no shadows
		//if !ShadowsOn {
		//	sep := widget.NewSeparator(root.UI)
		//	sep.Size.Y = SeparatorWidth
		//	sep.SetTheme(&UITheme.Toolbar)
		//	bgLayer.Append(sep)
		//	bgLayer.SetChildFill(sep, true, false)
		//}

		// columns
		root.Cols = NewColumns(root)
		bgLayer.Append(root.Cols)
	}
}

func (l *Root) OnChildMarked(child widget.Node, newMarks widget.Marks) {
	l.MultiLayer.OnChildMarked(child, newMarks)
	// dynamic toolbar
	if l.Toolbar != nil && l.Toolbar.Marks.HasAny(widget.MarkNeedsLayout) {
		l.BgLayer.MarkNeedsLayout()
	}
}

//----------

func (l *Root) GoodRowPos() *RowPos {

	var best struct {
		r       *image.Rectangle
		area    int
		col     *Column
		nextRow *Row
	}

	// default position if nothing better is found
	best.col = l.Cols.FirstChildColumn()

	for _, c := range l.Cols.Columns() {
		rows := c.Rows()

		// space before first row
		s := c.Bounds.Size()
		if len(rows) > 0 {
			s.Y = rows[0].Bounds.Min.Y - c.Bounds.Min.Y
		}
		a := s.X * s.Y
		if a > best.area {
			best.area = a
			best.col = c
			best.nextRow = nil
			if len(rows) > 0 {
				best.nextRow = rows[0]
			}
		}

		// space between rows
		for _, r := range rows {
			s := r.TextArea.Bounds.Size()
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

	return NewRowPos(best.col, best.nextRow)
}

//----------

type RowPos struct {
	Column  *Column
	NextRow *Row

	// TODO: percent for rowslayout.spl
}

func NewRowPos(col *Column, nextRow *Row) *RowPos {
	return &RowPos{col, nextRow}
}
