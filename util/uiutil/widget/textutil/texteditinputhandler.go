package textutil

import (
	"image"
	"unicode"

	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

type TextEditInputHandler struct {
	tex *widget.TextEditX
}

func NewTextEditInputHandler(tex *widget.TextEditX) *TextEditInputHandler {
	return &TextEditInputHandler{tex: tex}
}

func (eh *TextEditInputHandler) OnInputEvent(ev interface{}, p image.Point) event.Handled {
	h := eh.tex.TextHistory.HandleInputEvent(ev, p) // undo/redo shortcuts
	if h == event.HFalse {
		h = eh.handleInputEvent2(ev, p)
	}
	return h
}

func (eh *TextEditInputHandler) handleInputEvent2(ev0 interface{}, p image.Point) event.Handled {
	te := eh.tex.TextEdit

	switch ev := ev0.(type) {
	case *event.KeyDown:
		eh.onKeyDown(ev)

	case *event.MouseDown:
		switch ev.Button {
		case event.ButtonLeft:
			if ev.Mods.ClearLocks().Is(event.ModShift) {
				MoveCursorToPoint(te, &ev.Point, true)
			} else {
				MoveCursorToPoint(te, &ev.Point, false)
			}
		}

	case *event.MouseDragMove:
		if ev.Buttons.Has(event.ButtonLeft) {
			MoveCursorToPoint(te, &ev.Point, true)
		}
	case *event.MouseDragEnd:
		switch ev.Button {
		case event.ButtonLeft:
			MoveCursorToPoint(te, &ev.Point, true)
		}

	case *event.MouseClick:
		return eh.onMouseClick(ev)
	case *event.MouseDoubleClick:
		return eh.onMouseDoubleClick(ev)
	case *event.MouseTripleClick:
		return eh.onMouseTripleClick(ev)
	}
	return event.HFalse
}

//----------

func (eh *TextEditInputHandler) onMouseClick(ev *event.MouseClick) event.Handled {
	te := eh.tex.TextEdit
	switch ev.Button {
	case event.ButtonMiddle:
		MoveCursorToPoint(te, &ev.Point, false)
		Paste(te, event.CPIPrimary)
		return event.HTrue
	}
	return event.HFalse
}
func (eh *TextEditInputHandler) onMouseDoubleClick(ev *event.MouseDoubleClick) event.Handled {
	te := eh.tex.TextEdit
	switch ev.Button {
	case event.ButtonLeft:
		MoveCursorToPoint(te, &ev.Point, false)
		SelectWord(te)
		return event.HTrue
	}
	return event.HFalse
}
func (eh *TextEditInputHandler) onMouseTripleClick(ev *event.MouseTripleClick) event.Handled {
	te := eh.tex.TextEdit
	switch ev.Button {
	case event.ButtonLeft:
		MoveCursorToPoint(te, &ev.Point, false)
		SelectLine(te)
		return event.HTrue
	}
	return event.HFalse
}

//----------

func (eh *TextEditInputHandler) onKeyDown(ev *event.KeyDown) {
	te := eh.tex.TextEdit
	mcl := ev.Mods.ClearLocks()

	makeCursorVisible := func() {
		te.MakeIndexVisible(te.TextCursor.Index())
	}

	switch ev.KeySym {
	case event.KSymAltL,
		event.KSymAltGr,
		event.KSymShiftL,
		event.KSymShiftR,
		event.KSymControlL,
		event.KSymControlR,
		event.KSymCapsLock,
		event.KSymNumLock,
		event.KSymInsert,
		event.KSymPageUp,
		event.KSymPageDown,
		event.KSymEscape,
		event.KSymSuperL: // windows key
		// ignore these
	case event.KSymRight:
		switch {
		case mcl.Is(event.ModCtrl | event.ModShift):
			MoveCursorJumpRight(te, true)
		case mcl.Is(event.ModCtrl):
			MoveCursorJumpRight(te, false)
		case mcl.Is(event.ModShift):
			MoveCursorRight(te, true)
		default:
			MoveCursorRight(te, false)
		}
		makeCursorVisible()
	case event.KSymLeft:
		switch {
		case mcl.Is(event.ModCtrl | event.ModShift):
			MoveCursorJumpLeft(te, true)
		case mcl.Is(event.ModCtrl):
			MoveCursorJumpLeft(te, false)
		case mcl.Is(event.ModShift):
			MoveCursorLeft(te, true)
		default:
			MoveCursorLeft(te, false)
		}
		makeCursorVisible()
	case event.KSymUp:
		switch {
		case mcl.Is(event.ModCtrl | event.ModAlt):
			MoveLineUp(te)
		case mcl.HasAny(event.ModShift):
			MoveCursorUp(te, true)
		default:
			MoveCursorUp(te, false)
		}
		makeCursorVisible()
	case event.KSymDown:
		switch {
		case mcl.Is(event.ModCtrl | event.ModShift | event.ModAlt):
			DuplicateLines(te)
		case mcl.Is(event.ModCtrl | event.ModAlt):
			MoveLineDown(te)
		case mcl.HasAny(event.ModShift):
			MoveCursorDown(te, true)
		default:
			MoveCursorDown(te, false)
		}
		makeCursorVisible()
	case event.KSymHome:
		switch {
		case mcl.Is(event.ModCtrl | event.ModShift):
			StartOfString(te, true)
		case mcl.Is(event.ModCtrl):
			StartOfString(te, false)
		case mcl.Is(event.ModShift):
			StartOfLine(te, true)
		default:
			StartOfLine(te, false)
		}
		makeCursorVisible()
	case event.KSymEnd:
		switch {
		case mcl.Is(event.ModCtrl | event.ModShift):
			EndOfString(te, true)
		case mcl.Is(event.ModCtrl):
			EndOfString(te, false)
		case mcl.Is(event.ModShift):
			EndOfLine(te, true)
		default:
			EndOfLine(te, false)
		}
		makeCursorVisible()
	case event.KSymBackspace:
		Backspace(te)
		makeCursorVisible()
	case event.KSymDelete:
		Delete(te)
	case event.KSymReturn:
		AutoIndent(te)
		makeCursorVisible()
	case event.KSymTabLeft:
		TabLeft(te)
		makeCursorVisible()
	case event.KSymTab:
		switch {

		// using KSymTabLeft case, this still needed?
		case mcl.Is(event.ModShift):
			TabLeft(te)

		default:
			TabRight(te)
		}
		makeCursorVisible()
	case event.KSymSpace:
		// ensure space even if modifiers are present
		InsertString(te, " ")
		makeCursorVisible()
	default:
		switch {
		case mcl.Is(event.ModCtrl | event.ModShift):
			switch ev.KeySym {
			case event.KSymD:
				Uncomment(eh.tex)
			}
		case mcl.Is(event.ModCtrl):
			switch ev.KeySym {
			case event.KSymD:
				Comment(eh.tex)
			case event.KSymC:
				Copy(te)
			case event.KSymX:
				Cut(te)
			case event.KSymV:
				Paste(te, event.CPIClipboard)
			case event.KSymK:
				RemoveLines(te)
			case event.KSymA:
				SelectAll(te)
			}
		case ev.KeySym >= event.KSymF1 && ev.KeySym <= event.KSymF12:
			// do nothing
		case !unicode.IsPrint(ev.Rune):
			// do nothing
		default:
			InsertString(te, string(ev.Rune))
			makeCursorVisible()
		}
	}
}
