package ui

import (
	"image"

	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type ContextFloatBox struct {
	*widget.FloatBox
	Label *widget.Label
	root  *Root
	//Enabled bool // used externally
}

func NewContextFloatBox(root *Root) *ContextFloatBox {
	cfb := &ContextFloatBox{root: root}

	cfb.Label = widget.NewLabel(root.UI)
	cfb.Label.Text.SetStr("todo...")
	cfb.Label.Pad.Left = 5
	cfb.Label.Pad.Right = 5
	cfb.Label.Border.SetAll(1)

	// TODO: scrollarea
	//scrollArea := widget.NewScrollArea(l.UI, cfb.Label)
	//scrollArea.LeftScroll = ScrollbarLeft
	//scrollArea.ScrollWidth = ScrollbarWidth
	//border := widget.NewPad(root.UI, scrollArea)
	//border.Set(1)

	container := WrapInBottomShadowOrNone(root.UI, cfb.Label)

	cfb.FloatBox = widget.NewFloatBox(root.MultiLayer, root.MultiLayer.ContextLayer, container)
	cfb.FloatBox.Hide()

	return cfb
}

//----------

//----------

func (cfb *ContextFloatBox) OnInputEvent(ev interface{}, p image.Point) event.Handle {
	switch ev.(type) {
	case *event.KeyUp,
		*event.KeyDown:
		// let lower layers get events
		return event.NotHandled
	}
	return event.Handled
}
