package ui

import (
	"image"
	"unicode"

	"github.com/jmigpin/editor/util/drawutil/drawer4"
	"github.com/jmigpin/editor/util/evreg"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
	"github.com/jmigpin/editor/util/uiutil/widget/textutil"
)

type TextArea struct {
	*widget.TextEditX
	*textutil.TextEditInputHandler
	EvReg                       *evreg.Register
	SupportClickInsideSelection bool

	ui *UI
}

func NewTextArea(ui *UI) *TextArea {
	ta := &TextArea{ui: ui}
	ta.TextEditX = widget.NewTextEditX(ui, ui)
	ta.TextEditInputHandler = textutil.NewTextEditInputHandler(ta.TextEditX)

	ta.OnSetStr = ta.onSetStr
	ta.OnWriteOp = ta.onWriteOp
	ta.EvReg = evreg.NewRegister()

	return ta
}

//----------

func (ta *TextArea) onSetStr() {
	ev := &TextAreaSetStrEvent{ta}
	ta.EvReg.RunCallbacks(TextAreaSetStrEventId, ev)
}

func (ta *TextArea) onWriteOp(u *iorw.RWCallbackWriteOp) {
	ev := &TextAreaWriteOpEvent{ta, u}
	ta.EvReg.RunCallbacks(TextAreaWriteOpEventId, ev)
}

//----------

func (ta *TextArea) OnInputEvent(ev0 interface{}, p image.Point) event.Handled {
	h := ta.handleInputEvent2(ev0, p) // editor shortcuts first
	if h == event.HFalse {
		h = ta.TextEditInputHandler.OnInputEvent(ev0, p)
	}
	return h
}

func (ta *TextArea) handleInputEvent2(ev0 interface{}, p image.Point) event.Handled {
	switch ev := ev0.(type) {
	case *event.MouseClick:
		switch ev.Button {
		case event.ButtonRight:
			m := ev.Mods.ClearLocks()
			switch {
			case m.Is(event.ModCtrl):
				if ta.selAnnCurEv(ev.Point, TASelAnnTypePrint) {
					return event.HTrue
				}
			case m.Is(event.ModCtrl | event.ModShift):
				if ta.selAnnCurEv(ev.Point, TASelAnnTypePrintAllPrevious) {
					return event.HTrue
				}
			}
			if !ta.SupportClickInsideSelection || !ta.PointIndexInsideSelection(ev.Point) {
				textutil.MoveCursorToPoint(ta.TextEdit, &ev.Point, false)
			}
			i := ta.GetIndex(ev.Point)
			ev2 := &TextAreaCmdEvent{ta, i}
			ta.EvReg.RunCallbacks(TextAreaCmdEventId, ev2)
			return event.HTrue
		}
	case *event.MouseDown:
		switch ev.Button {
		case event.ButtonRight:
			ta.Cursor = event.PointerCursor
		case event.ButtonLeft:
			m := ev.Mods.ClearLocks()
			if m.Is(event.ModCtrl) {
				if ta.selAnnCurEv(ev.Point, TASelAnnTypeCurrent) {
					return event.HTrue
				}
			}
		case event.ButtonWheelUp:
			m := ev.Mods.ClearLocks()
			if m.Is(event.ModCtrl) {
				if ta.selAnnCurEv(ev.Point, TASelAnnTypeCurrentPrev) {
					return event.HTrue
				}
			}
		case event.ButtonWheelDown:
			m := ev.Mods.ClearLocks()
			if m.Is(event.ModCtrl) {
				if ta.selAnnCurEv(ev.Point, TASelAnnTypeCurrentNext) {
					return event.HTrue
				}
			}
		}
	case *event.MouseUp:
		switch ev.Button {
		case event.ButtonRight:
			ta.Cursor = event.NoneCursor
		}
	case *event.MouseDragStart:
		switch ev.Button {
		case event.ButtonRight:
			ta.Cursor = event.NoneCursor
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
	return event.HFalse
}

//----------

func (ta *TextArea) selAnnCurEv(p image.Point, typ TASelAnnType) bool {
	if d, ok := ta.Drawer.(*drawer4.Drawer); ok {
		if d.Opt.Annotations.On {
			i, o, ok := d.AnnotationsIndexOf(p)
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

func (ta *TextArea) inlineCompleteEv() event.Handled {
	if ta.TextCursor.SelectionOn() {
		return event.HFalse
	}

	// previous rune should not be a space
	offset := ta.TextCursor.Index()
	rw := ta.TextCursor.RW()
	ru, _, err := rw.ReadRuneAt(offset - 1)
	if err != nil {
		return event.HFalse
	}
	if unicode.IsSpace(ru) {
		return event.HFalse
	}

	ev2 := &TextAreaInlineCompleteEvent{ta, offset, false}
	ta.EvReg.RunCallbacks(TextAreaInlineCompleteEventId, ev2)
	return ev2.Handled
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

func (ta *TextArea) Layout() {
	ta.TextEditX.Layout()
	ta.setDrawer4Opts()
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
	TextAreaSetStrEventId = iota
	TextAreaWriteOpEventId
	TextAreaCmdEventId
	TextAreaSelectAnnotationEventId
	TextAreaInlineCompleteEventId
)

//----------

type TextAreaSetStrEvent struct {
	TextArea *TextArea
}
type TextAreaWriteOpEvent struct {
	TextArea *TextArea
	WriteOp  *iorw.RWCallbackWriteOp
}
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

type TASelAnnType int

const (
	TASelAnnTypeCurrent TASelAnnType = iota // make current
	TASelAnnTypeCurrentPrev
	TASelAnnTypeCurrentNext
	TASelAnnTypePrint
	TASelAnnTypePrintAllPrevious
)

//----------

type TextAreaInlineCompleteEvent struct {
	TextArea *TextArea
	Offset   int

	Handled event.Handled // allow callbacks to set value
}
