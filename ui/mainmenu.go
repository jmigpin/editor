package ui

import (
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type MainMenuButton struct {
	*widget.FloatButton
	Toolbar *Toolbar
}

func NewMainMenuButton(root *Root) *MainMenuButton {
	mmb := &MainMenuButton{}

	// create here just to add to the floatbutton instantiation
	content := &widget.ENode{}

	mmb.FloatButton = widget.NewFloatButton(root.UI, root.MultiLayer, content)
	mmb.FloatButton.Label.Text.SetStr(string(rune(8801))) // 3 lines rune
	mmb.FloatButton.Label.Pad.Left = 5
	mmb.FloatButton.Label.Pad.Right = 5

	// theme
	mmb.SetThemePaletteNamePrefix("mm_")
	content.SetThemePaletteNamePrefix("mm_content_")

	// float content
	mmb.Toolbar = NewToolbar(root.UI)
	pad := widget.NewPad(root.UI, mmb.Toolbar)
	pad.SetAll(10)
	border := widget.NewBorder(root.UI, pad)
	border.SetAll(1)
	n1 := WrapInBottomShadowOrNone(root.UI, border)
	content.Append(n1)

	return mmb
}
