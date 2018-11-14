package ui

import (
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

		// columns
		root.Cols = NewColumns(root)
		bgLayer.Append(root.Cols)
	}

	root.ContextFloatBox = NewContextFloatBox(root)
}

func (l *Root) OnChildMarked(child widget.Node, newMarks widget.Marks) {
	l.MultiLayer.OnChildMarked(child, newMarks)
	// dynamic toolbar
	if l.Toolbar != nil && l.Toolbar.HasAnyMarks(widget.MarkNeedsLayout) {
		l.BgLayer.MarkNeedsLayout()
	}
}
