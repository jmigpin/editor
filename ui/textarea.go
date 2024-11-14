package ui

import (
	"image"

	"github.com/jmigpin/editor/util/drawutil/drawer4"
	"github.com/jmigpin/editor/util/evreg"
	"github.com/jmigpin/editor/util/iout/iorw/rwedit"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type TextArea struct {
	*widget.TextEditX

	EvReg                       evreg.Register
	SupportClickInsideSelection bool

	ui *UI
}

func NewTextArea(ui *UI) *TextArea {
	ta := &TextArea{ui: ui}
	ta.TextEditX = widget.NewTextEditX(ui)
	return ta
}

//----------

func (ta *TextArea) OnInputEvent(ev0 any, p image.Point) event.Handled {
	h := event.Handled(false)

	// input events callbacks (terminal related)
	if !h {
		ev2 := &TextAreaInputEvent{TextArea: ta, Event: ev0}
		ta.EvReg.RunCallbacks(TextAreaInputEventId, ev2)
		h = ev2.ReplyHandled
	}

	// select annotation events
	if !h {
		h = ta.handleInputEvent2(ev0, p)
		// consider handled to avoid root events to select global annotations
		if h {
			return true
		}
	}

	if !h {
		h = ta.TextEditX.OnInputEvent(ev0, p)
		// don't consider handled to allow ui.Row to get inputevents
		if h {
			return false
		}
	}

	return h
}

func (ta *TextArea) handleInputEvent2(ev0 any, p image.Point) event.Handled {
	switch ev := ev0.(type) {
	case *event.MouseClick:
		switch ev.Button {
		case event.ButtonRight:
			m := ev.Mods.ClearLocks()
			switch {
			case m.Is(event.ModCtrl):
				if ta.selAnnCurEv(ev.Point, TasatPrint) {
					return true
				}
			case m.Is(event.ModCtrl | event.ModShift):
				if ta.selAnnCurEv(ev.Point, TasatPrintPreviousAll) {
					return true
				}
			}
			if !ta.SupportClickInsideSelection || !ta.PointIndexInsideSelection(ev.Point) {
				rwedit.MoveCursorToPoint(ta.EditCtx(), ev.Point, false)
			}
			i := ta.GetIndex(ev.Point)
			ev2 := &TextAreaCmdEvent{ta, i}
			ta.EvReg.RunCallbacks(TextAreaCmdEventId, ev2)
			return true
		}
	case *event.MouseDown:
		switch ev.Button {
		case event.ButtonRight:
			ta.ENode.Cursor = event.PointerCursor
		case event.ButtonLeft:
			m := ev.Mods.ClearLocks()
			if m.Is(event.ModCtrl) {
				if ta.selAnnCurEv(ev.Point, TasatMsg) {
					return true
				}
			}
		case event.ButtonWheelUp:
			m := ev.Mods.ClearLocks()
			if m.Is(event.ModCtrl) {
				if ta.selAnnCurEv(ev.Point, TasatMsgPrev) {
					return true
				}
			}
		case event.ButtonWheelDown:
			m := ev.Mods.ClearLocks()
			if m.Is(event.ModCtrl) {
				if ta.selAnnCurEv(ev.Point, TasatMsgNext) {
					return true
				}
			}
		}
	case *event.MouseUp:
		switch ev.Button {
		case event.ButtonRight:
			ta.ENode.Cursor = event.NoneCursor
		}
	case *event.MouseDragStart:
		switch ev.Button {
		case event.ButtonRight:
			ta.ENode.Cursor = event.NoneCursor
		}
	case *event.KeyDown:
		m := ev.Mods.ClearLocks()
		switch {
		case m.Is(event.ModNone):
			switch ev.KeySym {
			case event.KSymTab:
				return ta.inlineCompleteEv()
			}
		}
	}
	return false
}

//----------

func (ta *TextArea) selAnnCurEv(p image.Point, typ TASelAnnType) bool {
	if d, ok := ta.Drawer.(*drawer4.Drawer); ok {
		if d.Opt.Annotations.On {
			//i, o, ok := d.AnnotationsIndexOf(p)
			//if ok {
			//	ev2 := &TextAreaSelectAnnotationEvent{ta, i, o, typ}
			//	ta.EvReg.RunCallbacks(TextAreaSelectAnnotationEventId, ev2)
			//	return true
			//}

			ev2 := &TextAreaSelectAnnotationEvent{ta, 0, 0, typ}
			i, o, ok := d.AnnotationsIndexOf(p)
			if ok {
				ev2.AnnotationIndex = i
				ev2.Offset = o
			} else {
				// not in an annotation, switch the general prev/next
				switch typ {
				case TasatMsgPrev:
					ev2.Type = TasatPrev
				case TasatMsgNext:
					ev2.Type = TasatNext
				default:
					return false
				}
			}
			ta.EvReg.RunCallbacks(TextAreaSelectAnnotationEventId, ev2)
			return true
		}
	}
	return false
}

//func (ta *TextArea) selAnnEv(typ TASelAnnType) {
//	ev2 := &TextAreaSelectAnnotationEvent{ta, 0, 0, typ}
//	ta.EvReg.RunCallbacks(TextAreaSelectAnnotationEventId, ev2)
//}

//----------

func (ta *TextArea) inlineCompleteEv() event.Handled {
	c := ta.Cursor()
	if c.HaveSelection() {
		return false
	}

	ev2 := &TextAreaInlineCompleteEvent{ta, c.Index(), false}
	ta.EvReg.RunCallbacks(TextAreaInlineCompleteEventId, ev2)
	return ev2.ReplyHandled
}

//----------

func (ta *TextArea) PointIndexInsideSelection(p image.Point) bool {
	c := ta.Cursor()
	if s, e, ok := c.SelectionIndexes(); ok {
		i := ta.GetIndex(p)
		return i >= s && i < e
	}
	return false
}

//----------

func (ta *TextArea) Layout() {
	ta.TextEditX.Layout()
	ta.setDrawer4Opts()

	ev2 := &TextAreaLayoutEvent{TextArea: ta}
	ta.EvReg.RunCallbacks(TextAreaLayoutEventId, ev2)
}

func (ta *TextArea) setDrawer4Opts() {
	if d, ok := ta.Drawer.(*drawer4.Drawer); ok {
		// scale cursor based on lineheight
		w := 1
		u := d.LineHeight()
		u2 := int(float64(u) * 0.08)
		if u2 > 1 {
			w = u2
		}
		d.Opt.Cursor.AddedWidth = w

		// set startoffsetx based on cursor
		d.Opt.RuneReader.StartOffsetX = d.Opt.Cursor.AddedWidth * 2
	}
}

//----------

const (
	TextAreaCmdEventId = iota
	TextAreaSelectAnnotationEventId
	TextAreaInlineCompleteEventId
	TextAreaInputEventId
	TextAreaLayoutEventId
)

//----------

type TextAreaCmdEvent struct {
	TextArea *TextArea
	Index    int
}

//----------

type TextAreaSelectAnnotationEvent struct {
	TextArea        *TextArea
	AnnotationIndex int
	Offset          int // annotation string click offset
	Type            TASelAnnType
}

//----------

type TASelAnnType int

const (
	TasatPrev TASelAnnType = iota
	TasatNext
	TasatMsg
	TasatMsgPrev
	TasatMsgNext
	TasatPrint
	TasatPrintPreviousAll
)

//----------

type TextAreaInlineCompleteEvent struct {
	TextArea *TextArea
	Offset   int

	ReplyHandled event.Handled // allow callbacks to set value // ex: Allow input event (`tab` key press) to function normally if the inlinecomplete is not being handled (ex: no lsproto server is registered for a filename extension)
}

//----------

type TextAreaInputEvent struct {
	TextArea     *TextArea
	Event        any
	ReplyHandled event.Handled
}

//----------

type TextAreaLayoutEvent struct {
	TextArea *TextArea
}
