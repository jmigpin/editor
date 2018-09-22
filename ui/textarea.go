package ui

import (
	"image"

	"github.com/jmigpin/editor/util/drawutil/drawer3"
	"github.com/jmigpin/editor/util/evreg"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
	"github.com/jmigpin/editor/util/uiutil/widget/textutil"
)

type TextArea struct {
	*widget.TextEditX
	*textutil.TextEditInputHandler
	EvReg *evreg.Register

	ui *UI
}

func NewTextArea(ui *UI) *TextArea {
	ta := &TextArea{ui: ui}
	ta.TextEditX = widget.NewTextEditX(ui, ui)
	ta.TextEditInputHandler = textutil.NewTextEditInputHandler(ta.TextEditX)

	ta.OnSetStr = ta.onSetStr
	ta.EvReg = evreg.NewRegister()

	return ta
}

//----------

func (ta *TextArea) onSetStr() {
	ev := &TextAreaSetStrEvent{ta}
	ta.EvReg.RunCallbacks(TextAreaSetStrEventId, ev)
}

//----------

func (ta *TextArea) OnInputEvent(ev0 interface{}, p image.Point) event.Handle {
	h := ta.TextEditInputHandler.OnInputEvent(ev0, p)
	if h == event.NotHandled {
		h = ta.handleInputEvent2(ev0, p)
	}
	return h
}

func (ta *TextArea) handleInputEvent2(ev0 interface{}, p image.Point) event.Handle {
	switch ev := ev0.(type) {
	case *event.MouseClick:
		switch ev.Button {
		case event.ButtonRight:
			if !ta.PointIndexInsideSelection(ev.Point) {
				textutil.MoveCursorToPoint(ta.TextEdit, &ev.Point, false)
			}
			i := ta.GetIndex(ev.Point)
			ev2 := &TextAreaCmdEvent{ta, i}
			ta.EvReg.RunCallbacks(TextAreaCmdEventId, ev2)
			return event.Handled
		case event.ButtonLeft:
			// positional drawer annotations click
			if d, ok := ta.Drawer.(*drawer3.PosDrawer); ok {
				if d.Annotations.On() {
					i, o, ok := d.BoundsAnnotationsIndexOf(ev.Point)
					if ok {
						ev2 := &TextAreaAnnotationClickEvent{ta, i, o, ev.Button}
						ta.EvReg.RunCallbacks(TextAreaAnnotationClickEventId, ev2)
						return event.Handled
					}
				}
			}
		}

	case *event.MouseDown:
		switch ev.Button {
		case event.ButtonRight:
			ta.Cursor = widget.PointerCursor
		}
	case *event.MouseUp:
		switch ev.Button {
		case event.ButtonRight:
			ta.Cursor = widget.NoneCursor
		}
	case *event.MouseDragStart:
		switch ev.Button {
		case event.ButtonRight:
			ta.Cursor = widget.NoneCursor
		}
	}
	return event.NotHandled
}

//----------

func (ta *TextArea) PointIndexInsideSelection(p image.Point) bool {
	if ta.TextCursor.SelectionOn() {
		i := ta.GetIndex(p)
		s, e := ta.TextCursor.SelectionIndexes()
		return i >= s && i < e
	}
	return false
}

//----------

const (
	TextAreaSetStrEventId = iota
	TextAreaCmdEventId
	TextAreaAnnotationClickEventId
)

type TextAreaCmdEvent struct {
	TextArea *TextArea
	Index    int
}
type TextAreaSetStrEvent struct {
	TextArea *TextArea
}
type TextAreaAnnotationClickEvent struct {
	TextArea        *TextArea
	AnnotationIndex int
	Offset          int
	Button          event.MouseButton
}
