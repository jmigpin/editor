package ui

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/widget"
)

// User Interface root (top) node.
type Root struct {
	widget.MultiLayer
	UI              *UI
	Toolbar         *Toolbar
	MainMenuButton  *MainMenuButton
	ContextFloatBox *ContextFloatBox
	Cols            *Columns

	rowSepHandlesMark widget.Rectangle
	colSepHandlesMark widget.Rectangle
}

func SetupRoot(ui *UI) {
	root := &Root{UI: ui}
	root.UI.Root = root // complete circle reference: ui<->root
	root.UI.RootNode = root

	// Embed nodes have their wrapper nodes set when they are appended to another node. The root node is not appended to any other node, therefore it needs to be set here.
	root.SetWrapperForRootNode(root)

	root.init()
}
func (root *Root) init() {
	//  background layer
	bgLayer := widget.NewBoxLayout()
	bgLayer.YAxis = true
	root.Append(bgLayer)

	// column/row layer marks to be able to insert in a specific order
	root.rowSepHandlesMark.SetHidden(true)
	root.Append(&root.rowSepHandlesMark)
	root.colSepHandlesMark.SetHidden(true)
	root.Append(&root.colSepHandlesMark)

	// context floatbox layer
	root.ContextFloatBox = NewContextFloatBox(root)
	root.Append(root.ContextFloatBox)

	// main-menu-button floatmenu layer
	mmb := NewMainMenuButton(root.UI)
	root.MainMenuButton = mmb
	root.Append(mmb.FloatMenu)

	// setup background layer after other layers are set
	{
		// top toolbar
		{
			ttb := widget.NewBoxLayout()
			bgLayer.Append(ttb)

			// toolbar
			root.Toolbar = NewToolbar(root.UI, bgLayer)
			ttb.Append(root.Toolbar)
			ttb.SetChildFlex(root.Toolbar, true, false)

			// main menu button
			mmb.Label.Border.Left = 1
			ttb.Append(mmb)
			ttb.SetChildFill(mmb, false, true)
		}

		// separator if there are no shadows
		if !ShadowsOn {
			sep := widget.NewSeparator(root.UI)
			sep.Size.Y = SeparatorWidth
			sep.SetTheme(&UITheme.Toolbar)
			bgLayer.Append(sep)
			bgLayer.SetChildFill(sep, true, false)
		}

		// columns
		root.Cols = NewColumns(root)
		bgLayer.Append(root.Cols)
	}
}

func (l *Root) InsertRowSepHandle(n widget.Node) {
	l.InsertBefore(n, &l.rowSepHandlesMark)
}
func (l *Root) InsertColSepHandle(n widget.Node) {
	l.InsertBefore(n, &l.colSepHandlesMark)
}

func (l *Root) GoodColumnRowPlace() (*Column, *Row) {

	// TODO: accept optional row, or take into consideration active row
	// TODO: don't go too far away, stay close (active row)
	// TODO: belongs in Columns?

	var best struct {
		r       *image.Rectangle
		area    int
		col     *Column
		nextRow *Row
	}

	// allow to find a column at ui start when the area is 0
	best.area = -1

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
