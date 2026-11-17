package ui

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type ActionMenuAction struct {
	Label string
	Run   func()
}

type ActionMenu struct {
	*widget.FloatBox
	root    *Root
	actions *widget.BoxLayout
}

func NewActionMenu(root *Root) *ActionMenu {
	am := &ActionMenu{root: root, actions: widget.NewBoxLayout()}
	am.actions.YAxis = true
	border := widget.NewBorder(root.UI, am.actions)
	border.SetAll(1)
	am.FloatBox = widget.NewFloatBox(root.MultiLayer, border)
	am.SetThemePaletteNamePrefix("mm_")
	root.MultiLayer.MenuLayer.Append(am.FloatBox)
	am.Hide()
	return am
}

func (am *ActionMenu) ShowAt(point image.Point, actions []ActionMenuAction) {
	am.Hide()
	old := []widget.Node{}
	am.actions.IterateWrappers2(func(child widget.Node) {
		old = append(old, child)
	})
	for _, child := range old {
		am.actions.Remove(child)
	}
	for _, action := range actions {
		action := action
		b := widget.NewButton(am.root.UI)
		b.Label.Text.SetStr(action.Label)
		b.Label.Pad.SetAll(5)
		b.OnClick = func(*event.MouseClick) {
			am.Hide()
			action.Run()
		}
		am.actions.Append(b)
		am.actions.SetChildFill(b, true, false)
	}
	am.RefPoint = point
	am.Show()
}

func (am *ActionMenu) AutoClose(ev any, point image.Point) {
	if !am.Visible() {
		return
	}
	if !point.In(am.Bounds) {
		switch ev.(type) {
		case *event.MouseDown:
			am.Hide()
		}
	}
}
