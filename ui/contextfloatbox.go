package ui

import (
	"image"

	"github.com/jmigpin/editor/util/drawutil/drawer4"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type ContextFloatBox struct {
	*widget.FloatBox

	root     *Root
	sa       *widget.ScrollArea
	TextArea *TextArea

	visibleOnAutoClose bool
}

func NewContextFloatBox(root *Root) *ContextFloatBox {
	cfb := &ContextFloatBox{root: root}

	cfb.TextArea = NewTextArea(root.UI)
	cfb.SetStr("")
	if d, ok := cfb.TextArea.Drawer.(*drawer4.Drawer); ok {
		//d.Opt.LineWrap.On = false
		d.Opt.RuneReader.StartOffsetX = 2 // covers cursor pixel
	}

	cfb.sa = widget.NewScrollArea(root.UI, cfb.TextArea, false, true)
	cfb.sa.LeftScroll = ScrollBarLeft

	border := widget.NewBorder(root.UI, cfb.sa)
	border.SetAll(1)

	container := WrapInBottomShadowOrNone(root.UI, border)

	cfb.FloatBox = widget.NewFloatBox(root.MultiLayer, container)
	cfb.FloatBox.MaxSize = image.Point{550, 100000}
	root.MultiLayer.ContextLayer.Append(cfb)
	cfb.FloatBox.Hide()

	cfb.SetThemePaletteNamePrefix("contextfloatbox_")

	return cfb
}

//----------

func (cfb *ContextFloatBox) SetStr(s string) {
	if s == "" {
		s = "No content provided."
	}
	cfb.TextArea.SetStr(s)
}

//----------

func (cfb *ContextFloatBox) Layout() {
	sw := UIThemeUtil.GetScrollBarWidth(cfb.TextArea.TreeThemeFont())
	cfb.sa.ScrollWidth = sw //* 2 / 3
	cfb.FloatBox.Layout()
}

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

//----------

func (cfb *ContextFloatBox) AutoClose(ev interface{}, p image.Point) {
	cfb.visibleOnAutoClose = cfb.Visible()
	if cfb.Visible() && !p.In(cfb.Bounds) {
		switch ev.(type) {
		case *event.KeyDown,
			*event.MouseDown:
			cfb.Hide()
			return
		case *event.MouseMove:
		default:
			//fmt.Printf("%T\n", ev)
		}
	}
}

//----------

func (cfb *ContextFloatBox) Toggle() {
	visible := cfb.Visible() || cfb.visibleOnAutoClose
	if !visible {
		cfb.Show()
	} else {
		cfb.Hide()
	}
}

//----------

func (cfb *ContextFloatBox) SetRefPointToTextAreaCursor(ta *TextArea) {
	p := ta.GetPoint(ta.TextCursor.Index())
	p.Y += ta.LineHeight()
	cfb.RefPoint = p
	// compensate scrollwidth for a better position
	if cfb.sa.LeftScroll {
		cfb.RefPoint.X -= cfb.sa.ScrollWidth
	}
}

//----------

func (cfb *ContextFloatBox) FindTextAreaUnderPointer() (*TextArea, bool) {
	// pointer position
	p, err := cfb.root.UI.QueryPointer()
	if err != nil {
		return nil, false
	}
	ta := cfb.visitToFindTA(*p, cfb.root)
	return ta, ta != nil
}

func (cfb *ContextFloatBox) visitToFindTA(p image.Point, node widget.Node) (ta *TextArea) {
	if p.In(node.Embed().Bounds) {
		if u, ok := node.(*TextArea); ok {
			return u
		}
		if u, ok := node.(*Toolbar); ok {
			return u.TextArea
		}
		if u, ok := node.(*RowToolbar); ok {
			return u.TextArea
		}
	}
	node.Embed().IterateWrappersReverse(func(n widget.Node) bool {
		u := cfb.visitToFindTA(p, n)
		if u != nil {
			ta = u
			return false
		}
		return true
	})
	return ta
}
