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
			m := ev.Mods.ClearLocks()
			if m.Is(event.ModCtrl) {
				if ta.selAnnCurEv(ev.Point, TASelAnnTypePrint) {
					return event.Handled
				}
			}
			if !ta.PointIndexInsideSelection(ev.Point) {
				textutil.MoveCursorToPoint(ta.TextEdit, &ev.Point, false)
			}
			i := ta.GetIndex(ev.Point)
			ev2 := &TextAreaCmdEvent{ta, i}
			ta.EvReg.RunCallbacks(TextAreaCmdEventId, ev2)
			return event.Handled
		}
	case *event.MouseDown:
		switch ev.Button {
		case event.ButtonRight:
			ta.Cursor = widget.PointerCursor
		case event.ButtonLeft:
			m := ev.Mods.ClearLocks()
			if m.Is(event.ModCtrl) {
				if ta.selAnnCurEv(ev.Point, TASelAnnTypeCurrent) {
					return event.Handled
				}
			}
		case event.ButtonWheelUp:
			m := ev.Mods.ClearLocks()
			if m.Is(event.ModCtrl) {
				if ta.selAnnCurEv(ev.Point, TASelAnnTypeCurrentPrev) {
					return event.Handled
				} else {
					ta.selAnnEv(TASelAnnTypePrev)
					return event.Handled
				}
			}
		case event.ButtonWheelDown:
			m := ev.Mods.ClearLocks()
			if m.Is(event.ModCtrl) {
				if ta.selAnnCurEv(ev.Point, TASelAnnTypeCurrentNext) {
					return event.Handled
				} else {
					ta.selAnnEv(TASelAnnTypeNext)
					return event.Handled
				}
			}
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
	case *event.KeyDown:
		m := ev.Mods.ClearLocks()
		if m.Is(event.ModCtrl) {
			switch ev.KeySym {
			case event.KSymF5:
				ta.selAnnEv(TASelAnnTypeLast)
				return event.Handled
			case event.KSymF9:
				ta.selAnnEv(TASelAnnTypeClear)
				return event.Handled
			}
		}
	}
	return event.NotHandled
}

//----------

func (ta *TextArea) selAnnCurEv(p image.Point, typ TASelAnnType) bool {
	if d, ok := ta.Drawer.(*drawer3.PosDrawer); ok {
		if d.Annotations.On() {
			i, o, ok := d.BoundsAnnotationsIndexOf(p)
			if ok {
				ev2 := &TextAreaSelectAnnotationEvent{ta, i, o, typ}
				ta.EvReg.RunCallbacks(TextAreaSelectAnnotationEventId, ev2)
				return true
			}
		}
	}
	return false
}
func (ta *TextArea) selAnnEv(typ TASelAnnType) {
	ev2 := &TextAreaSelectAnnotationEvent{ta, 0, 0, typ}
	ta.EvReg.RunCallbacks(TextAreaSelectAnnotationEventId, ev2)
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
	TextAreaSelectAnnotationEventId
)

type TextAreaCmdEvent struct {
	TextArea *TextArea
	Index    int
}
type TextAreaSetStrEvent struct {
	TextArea *TextArea
}

//----------

type TextAreaSelectAnnotationEvent struct {
	TextArea        *TextArea
	AnnotationIndex int
	Offset          int // annotation string click offset
	Type            TASelAnnType
}

type TASelAnnType int

const (
	TASelAnnTypeCurrent TASelAnnType = iota // make current
	TASelAnnTypeCurrentPrev
	TASelAnnTypeCurrentNext
	TASelAnnTypePrev
	TASelAnnTypeNext
	TASelAnnTypeLast
	TASelAnnTypeClear
	TASelAnnTypePrint
)

//----------
