package core

import (
	"fmt"
	"io"

	"github.com/jmigpin/editor/core/termemu"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/evreg"
	"github.com/jmigpin/editor/util/uiutil/event"
)

// textarea read closer; passes to the terminal emulator keyboard/mouse events
type ERowTaReadCloser struct {
	erow *ERow
	reg  *evreg.Regist

	pr *io.PipeReader
	pw *io.PipeWriter
}

func newERowTaReadCloser(erow *ERow) *ERowTaReadCloser {
	tarc := &ERowTaReadCloser{erow: erow}

	// register to handle textarea input events
	ta := tarc.erow.Row.TextArea
	tarc.reg = ta.EvReg.Add(ui.TextAreaInputEventId, tarc.onTextAreaInputEvent)
	// setup pipe to write for reading
	tarc.pr, tarc.pw = io.Pipe()

	return tarc
}

func (tarc *ERowTaReadCloser) isOn() bool {
	u := tarc.erow.runOpts
	return u.forwardKb || u.forwardMouse
}

//----------

func (tarc *ERowTaReadCloser) Read(p []byte) (int, error) {
	return tarc.pr.Read(p)
}

func (tarc *ERowTaReadCloser) Close() error {
	tarc.reg.Unregister()

	_ = tarc.pr.Close()
	return tarc.pw.Close()
}

//----------

func (tarc *ERowTaReadCloser) writeToRead(s string) error {
	_, err := tarc.pw.Write([]byte(s))
	if err != nil {
		tarc.erow.Ed.Errorf("textarea.writeToRead: %w", err)
		return err
	}
	return nil
}
func (tarc *ERowTaReadCloser) writePasteToRead(s string) error {
	if tarc.bracketedPaste() {
		s = bracketedPaste(s)
	}
	return tarc.writeToRead(s)
}

//----------

func (tarc *ERowTaReadCloser) bracketedPaste() bool {
	u := tarc.erow.optTemu
	return u != nil && u.emu.ScrPrivModes().BracketedPaste()
}
func (tarc *ERowTaReadCloser) lineFeedNewline() bool {
	u := tarc.erow.optTemu
	return u != nil && u.emu.ScrPrivModes().LineFeedNewline()
}
func (tarc *ERowTaReadCloser) appCursorKeys() bool {
	u := tarc.erow.optTemu
	return u != nil && u.emu.ScrPrivModes().AppCursorKeys()
}

//----------

func (tarc *ERowTaReadCloser) onTextAreaInputEvent(ev0 any) {
	if !tarc.isOn() {
		return
	}

	ev1 := ev0.(*ui.TextAreaInputEvent)

	switch ev2 := ev1.Event.(type) {
	case *event.KeyDown:
		if !tarc.erow.runOpts.forwardKb {
			break
		}

		if ok := tarc.kbCopyingWarning(ev1, ev2); ok {
			return
		}
		if ok := tarc.kbPaste(ev1, ev2); ok {
			return
		}
		if ok := tarc.kbEncode(ev1, ev2); ok {
			return
		}

	case *event.MouseClick:
		if !tarc.erow.runOpts.forwardMouse {
			break
		}

		if ok := tarc.mousePaste(ev1, ev2); ok {
			return
		}
	}
}

//----------

func (tarc *ERowTaReadCloser) kbPaste(ev1 *ui.TextAreaInputEvent, ev2 *event.KeyDown) bool {
	// support pasting (ctrl+v)
	if ev2.KeySym != event.KSymV {
		return false
	}
	mcl := ev2.Mods.ClearLocks()
	if !mcl.Is(event.ModCtrl) {
		return false
	}

	ev1.TextArea.EditCtx().Fns.GetClipboardData(event.CIClipboard, func(s string, err error) {
		if err != nil {
			return
		}
		_ = tarc.writePasteToRead(s)
	})
	// handled
	ev1.ReplyHandled = event.Handled(true)
	return true
}

func (tarc *ERowTaReadCloser) mousePaste(ev1 *ui.TextAreaInputEvent, ev2 *event.MouseClick) bool {
	// support pasting (middle click)
	if ev2.Button != event.ButtonMiddle {
		return false
	}
	mcl := ev2.Mods.ClearLocks()
	if !mcl.Is(0) {
		return false
	}
	ev1.TextArea.EditCtx().Fns.GetClipboardData(event.CIPrimary, func(s string, err error) {
		if err != nil {
			return
		}
		_ = tarc.writePasteToRead(s)
	})
	// handled
	ev1.ReplyHandled = event.Handled(true)
	return true
}

func (tarc *ERowTaReadCloser) kbCopyingWarning(ev1 *ui.TextAreaInputEvent, ev2 *event.KeyDown) bool {
	// warn about keys going to the exec instead of copying (ctrl+c)
	if ev2.KeySym != event.KSymC {
		return false
	}
	mcl := ev2.Mods.ClearLocks()
	if !mcl.Is(event.ModCtrl) {
		return false
	}

	// there must be a selection for the warn to show (less annoying)
	_, _, ok := ev1.TextArea.Cursor().SelectionIndexes()
	if !ok {
		return false
	}

	err := fmt.Errorf("warning: the keyboard input is being redirected to the executable, therefore your Ctrl+C is not copying, use the mouse to copy/paste on select")
	//fmt.Println(err)
	ev1.TextArea.EditCtx().Fns.Error(err) // TODO: get to ed.error?

	return true
}

//----------

func (tarc *ERowTaReadCloser) kbEncode(ev1 *ui.TextAreaInputEvent, ev2 *event.KeyDown) bool {
	s := tarc.kbEncodeToStr(ev1, ev2)
	if s != "" {
		if err := tarc.writeToRead(s); err != nil {
			return false
		}
		// handled
		ev1.ReplyHandled = event.Handled(true)
		return true
	}
	return false
}
func (tarc *ERowTaReadCloser) kbEncodeToStr(ev1 *ui.TextAreaInputEvent, ev2 *event.KeyDown) string {
	mods := normalizeTermKeyMods(ev2.Mods)

	encodeEsc := func(s string) string {
		mods, ok := encodeKeyMods(mods)
		if ok {
			return termemu.SeqEscCsi + "1;" + mods + s
		}
		return tarc.encodeEsc(s)
	}

	switch ev2.KeySym {
	case event.KSymAltL,
		event.KSymAltR,
		event.KSymAltGr,
		event.KSymShiftL,
		event.KSymShiftR,
		event.KSymControlL,
		event.KSymControlR,
		event.KSymSuperL,
		event.KSymSuperR,
		event.KSymCapsLock,
		event.KSymNumLock:
		return ""

	case event.KSymReturn, event.KSymKeypadEnter:
		s := "\r"
		if mods.HasAny(event.ModAlt) {
			s = "\x1b" + s
		}
		return s

	case event.KSymBackspace:
		s := "\b"
		if tarc.lineFeedNewline() {
			s = string('\x7f') // del
		}
		if mods.HasAny(event.ModAlt) {
			s = "\x1b" + s
		}
		return s

	case event.KSymSpace:
		if mods.HasAny(event.ModCtrl) {
			return "\x00"
		}
		s := " "
		if mods.HasAny(event.ModAlt) {
			s = "\x1b" + s
		}
		return s

	case event.KSymUp:
		return encodeEsc("A")
	case event.KSymDown:
		return encodeEsc("B")
	case event.KSymRight:
		return encodeEsc("C")
	case event.KSymLeft:
		return encodeEsc("D")

	case event.KSymHome:
		return encodeEsc("H")
	case event.KSymEnd:
		return encodeEsc("F")

	case event.KSymInsert:
		return termemu.SeqEscCsi + "2~"
	case event.KSymDelete:
		return termemu.SeqEscCsi + "3~"
	case event.KSymPageUp:
		return termemu.SeqEscCsi + "5~"
	case event.KSymPageDown:
		return termemu.SeqEscCsi + "6~"

	case event.KSymEscape:
		return string('\x1b')
	case event.KSymTab:
		s := "\t"
		if mods.HasAny(event.ModAlt) {
			s = "\x1b" + s
		}
		return s

	default:
		s := string(ev2.Rune)
		if mods.HasAny(event.ModCtrl) {
			if ev2.Rune <= 0x7f {
				s = string(encodeCtrl(byte(ev2.Rune)))
			}
		}
		if mods.HasAny(event.ModAlt) {
			s = "\x1b" + s
		}

		// ignore
		if ev2.Rune >= 0xff00 && ev2.Rune <= 0xffff {
			return ""
		}

		return s
	}
}

func normalizeTermKeyMods(km event.KeyModifiers) event.KeyModifiers {
	km = km.ClearLocks()
	if km.HasAny(event.ModAltGr) {
		// AltGr is a keyboard-layout selector, not terminal Meta/Ctrl.
		km &^= event.ModAltGr | event.ModCtrl | event.ModAlt
	}
	return km
}

//----------

func (tarc *ERowTaReadCloser) encodeEsc(seq string) string {
	if tarc.appCursorKeys() {
		return termemu.SeqEscO + seq
	}
	return termemu.SeqEscCsi + seq
}

//----------
//----------
//----------

func encodeCtrl(b byte) byte {
	if b == '?' { // // special case: Ctrl+? => DEL
		return 0x7F
	}
	return b & 0x1F // clears case bit; A/a -> 0x01, etc.
}

func bracketedPaste(s string) string {
	open := termemu.SeqEscCsi + "200~"
	close := termemu.SeqEscCsi + "201~"
	return open + s + close
}

func encodeKeyMods(km event.KeyModifiers) (string, bool) {
	mod := ""
	switch {
	case km.Is(event.ModShift):
		mod = "2"
	case km.Is(event.ModAlt):
		mod = "3"
	case km.Is(event.ModShift | event.ModAlt):
		mod = "4"
	case km.Is(event.ModCtrl):
		mod = "5"
	case km.Is(event.ModShift | event.ModCtrl):
		mod = "6"
	case km.Is(event.ModAlt | event.ModCtrl):
		mod = "7"
	case km.Is(event.ModShift | event.ModAlt | event.ModCtrl):
		mod = "8"
	}
	return mod, mod != ""
}
