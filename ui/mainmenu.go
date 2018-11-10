package ui

import (
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type MainMenuButton struct {
	*widget.FloatBoxButton
	Toolbar *Toolbar
}

func NewMainMenuButton(root *Root) *MainMenuButton {
	mmb := &MainMenuButton{}

	content := &widget.ENode{}

	mmb.FloatBoxButton = widget.NewFloatBoxButton(root.UI, root.MultiLayer, root.MenuLayer, content)
	mmb.FloatBoxButton.Label.Text.SetStr(string(rune(8801))) // 3 lines rune
	mmb.FloatBoxButton.Label.Pad.Left = 5
	mmb.FloatBoxButton.Label.Pad.Right = 5

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
