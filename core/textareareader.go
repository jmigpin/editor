package core

import (
	"io"

	"github.com/jmigpin/editor/core/termemu"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/evreg"
	"github.com/jmigpin/editor/util/uiutil/event"
)

// used as a reader to pass to the terminal emulator for input like keyboard/mouse events
type TextAreaReader struct {
	handleKeybInput bool

	reg  *evreg.Regist
	temu *termemu.Emu

	pr *io.PipeReader
	pw *io.PipeWriter
}

func newTextareaReader(ta *ui.TextArea) *TextAreaReader {
	tard := &TextAreaReader{}

	tard.pr, tard.pw = io.Pipe()

	// register to handle textarea input events
	tard.reg = ta.EvReg.Add(ui.TextAreaInputEventId, tard.onTextAreaInputEvent)

	return tard
}

//----------

func (tard *TextAreaReader) Read(p []byte) (int, error) {
	return tard.pr.Read(p)
}
func (tard *TextAreaReader) Close() error {
	defer tard.reg.Unregister()
	_ = tard.pr.Close()
	return tard.pw.Close()
}

//----------

func (tard *TextAreaReader) onTextAreaInputEvent(ev0 any) {
	ev := ev0.(*ui.TextAreaInputEvent)

	b, ok := tard.eventToBytes(ev.Event)
	if ok {
		_, err := tard.pw.Write(b)
		_ = err // TODO
	}
	ev.ReplyHandled = event.Handled(ok) // let events bubble up
}
func (tard *TextAreaReader) eventToBytes(ev any) ([]byte, bool) {
	switch t := ev.(type) {
	case *event.KeyDown:
		if tard.handleKeybInput {
			return tard.keydownToBytes(t)
		}

		//case *event.KeyUp:
		// TODO
	}
	return nil, false
}

//----------

func (tard *TextAreaReader) keydownToBytes(ev *event.KeyDown) ([]byte, bool) {
	s := tard.keydownToString(ev)
	return []byte(s), len(s) != 0
}
func (tard *TextAreaReader) keydownToString(ev *event.KeyDown) string {

	encodeEsc := func(s string) string {
		//mods, ok := encodeKeyMods(ev.Mods)
		//if ok {
		//	s = "1;" + mods + s
		//}
		return tard.encodeEsc(s)
	}

	switch ev.KeySym {
	case event.KSymReturn, event.KSymKeypadEnter:
		//ckm := tard.temu.ScrMode().CursorKeysMode()
		//if ckm {
		//	return encodeEsc("M")
		//}
		m := tard.temu.ScrMode().LineFeedNewlineMode()
		if m {
			// introduces extra newlines: aptitude
			//return []byte("\r\n"), true
		}
		return "\r"

	case event.KSymBackspace:
		m := tard.temu.ScrMode().LineFeedNewlineMode()
		if m {
			return string(0x7f) // del
		}
		return "\b"

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

	//case event.KSymHome:
	//	return seqEscCsi + "1~"
	case event.KSymInsert:
		return seqEscCsi + "2~"
	case event.KSymDelete:
		return seqEscCsi + "3~"
	//case event.KSymEnd:
	//	return seqEscCsi + "4~"
	case event.KSymPageUp:
		return seqEscCsi + "5~"
	case event.KSymPageDown:
		return seqEscCsi + "6~"
	//case event.KSymHome:
	//	return seqEscCsi + "7~"
	//case event.KSymEnd:
	//	return seqEscCsi + "8~"

	case event.KSymEscape:
		return string(27)
	case event.KSymTab:
		return "\t"

	default:

		if ev.Mods.HasAny(event.ModCtrl) {
			if ev.Rune <= 0x7f {
				return string(encodeCtrl(byte(ev.Rune)))
			}
		}

		// ignore
		if ev.Rune >= 0xff00 && ev.Rune <= 0xffff {
			return ""
		}

		return string(ev.Rune)
	}
}

//----------

func (tard *TextAreaReader) encodeEsc(seq string) string {
	ckm := tard.temu.ScrMode().CursorKeysMode()
	if ckm {
		return seqEscO + seq
	}
	// normal mode
	return seqEscCsi + seq
}

//----------
//----------
//----------

const seqEsc = "\x1b"
const seqEscCsi = seqEsc + "["
const seqEscO = seqEsc + "O"

func encodeCtrl(b byte) byte {
	if b == '?' { // // special case: Ctrl+? => DEL
		return 0x7F
	}
	return b & 0x1F // clears case bit; A/a -> 0x01, etc.
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
