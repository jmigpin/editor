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

func (eh *TextEditInputHandler) OnInputEvent(ev interface{}, p image.Point) event.Handle {
	h := eh.tex.TextHistory.HandleInputEvent(ev, p) // undo/redo shortcuts
	if h == event.NotHandled {
		h = eh.handleInputEvent2(ev, p)
	}
	return h
}

func (eh *TextEditInputHandler) handleInputEvent2(ev0 interface{}, p image.Point) event.Handle {
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
			// TODO: make cursor visible?
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
	return event.NotHandled
}

//----------

func (eh *TextEditInputHandler) onMouseClick(ev *event.MouseClick) event.Handle {
	te := eh.tex.TextEdit
	switch ev.Button {
	case event.ButtonMiddle:
		MoveCursorToPoint(te, &ev.Point, false)
		Paste(te, event.CPIPrimary)
		return event.Handled
	}
	return event.NotHandled
}
func (eh *TextEditInputHandler) onMouseDoubleClick(ev *event.MouseDoubleClick) event.Handle {
	te := eh.tex.TextEdit
	switch ev.Button {
	case event.ButtonLeft:
		MoveCursorToPoint(te, &ev.Point, false)
		SelectWord(te)
		return event.Handled
	}
	return event.NotHandled
}
func (eh *TextEditInputHandler) onMouseTripleClick(ev *event.MouseTripleClick) event.Handle {
	te := eh.tex.TextEdit
	switch ev.Button {
	case event.ButtonLeft:
		MoveCursorToPoint(te, &ev.Point, false)
		SelectLine(te)
		return event.Handled
	}
	return event.NotHandled
}

//----------

func (eh *TextEditInputHandler) onKeyDown(ev *event.KeyDown) {
	te := eh.tex.TextEdit

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
		makeCursorVisible()       // make a partially visible cursor visible
		defer makeCursorVisible() // adjust adjacent lines just one line instead of centralizing

		switch {
		case ev.Mods.ClearLocks().Is(event.ModCtrl | event.ModShift):
			MoveCursorJumpRight(te, true)
		case ev.Mods.ClearLocks().Is(event.ModCtrl):
			MoveCursorJumpRight(te, false)
		case ev.Mods.ClearLocks().Is(event.ModShift):
			MoveCursorRight(te, true)
		default:
			MoveCursorRight(te, false)
		}
	case event.KSymLeft:
		makeCursorVisible()
		defer makeCursorVisible()

		switch {
		case ev.Mods.ClearLocks().Is(event.ModCtrl | event.ModShift):
			MoveCursorJumpLeft(te, true)
		case ev.Mods.ClearLocks().Is(event.ModCtrl):
			MoveCursorJumpLeft(te, false)
		case ev.Mods.ClearLocks().Is(event.ModShift):
			MoveCursorLeft(te, true)
		default:
			MoveCursorLeft(te, false)
		}
	case event.KSymUp:
		makeCursorVisible()
		defer makeCursorVisible()

		switch {
		case ev.Mods.ClearLocks().Is(event.ModCtrl | event.ModAlt):
			MoveLineUp(te)
		case ev.Mods.ClearLocks().HasAny(event.ModShift):
			MoveCursorUp(te, true)
		default:
			MoveCursorUp(te, false)
		}
	case event.KSymDown:
		makeCursorVisible()
		defer makeCursorVisible()

		switch {
		case ev.Mods.ClearLocks().Is(event.ModCtrl | event.ModShift | event.ModAlt):
			DuplicateLines(te)
		case ev.Mods.ClearLocks().Is(event.ModCtrl | event.ModAlt):
			MoveLineDown(te)
		case ev.Mods.ClearLocks().HasAny(event.ModShift):
			MoveCursorDown(te, true)
		default:
			MoveCursorDown(te, false)
		}
	case event.KSymHome:
		switch {
		case ev.Mods.ClearLocks().Is(event.ModCtrl | event.ModShift):
			StartOfString(te, true)
		case ev.Mods.ClearLocks().Is(event.ModCtrl):
			StartOfString(te, false)
		case ev.Mods.ClearLocks().Is(event.ModShift):
			StartOfLine(te, true)
		default:
			StartOfLine(te, false)
		}
		makeCursorVisible()
	case event.KSymEnd:
		switch {
		case ev.Mods.ClearLocks().Is(event.ModCtrl | event.ModShift):
			EndOfString(te, true)
		case ev.Mods.ClearLocks().Is(event.ModCtrl):
			EndOfString(te, false)
		case ev.Mods.ClearLocks().Is(event.ModShift):
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
		case ev.Mods.ClearLocks().Is(event.ModShift):
			TabLeft(te)

		default:
			TabRight(te)
		}
		makeCursorVisible()
	case ' ':
		// ensure space even if modifiers are present
		InsertString(te, " ")
		makeCursorVisible()
	default:
		switch {
		case ev.KeySym >= event.KSymF1 && ev.KeySym <= event.KSymF12:
			// do nothing
		case !unicode.IsPrint(ev.Rune):
			// do nothing
		case ev.Mods.ClearLocks().Is(event.ModCtrl | event.ModShift):
			switch ev.LowerRune() {
			case 'd':
				Uncomment(eh.tex)
			}
		case ev.Mods.ClearLocks().Is(event.ModCtrl):
			switch ev.LowerRune() {
			case 'd':
				Comment(eh.tex)
			case 'c':
				Copy(te)
			case 'x':
				Cut(te)
			case 'v':
				Paste(te, event.CPIClipboard)
			case 'k':
				RemoveLines(te)
			case 'a':
				SelectAll(te)
			}
		default:
			InsertString(te, string(ev.Rune))
			makeCursorVisible()
		}
	}
}
