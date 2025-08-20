package core

import (
	"io"

	"github.com/jmigpin/editor/core/termemu"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/evreg"
	"github.com/jmigpin/editor/util/uiutil/event"
)

//godebug:annotatefile

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
	ev.ReplyHandled = event.Handled(ok)
}
func (tard *TextAreaReader) eventToBytes(ev any) ([]byte, bool) {
	switch t := ev.(type) {
	case *event.KeyDown:
		if tard.handleKeybInput {
			return tard.kdToBytes(t)
		}

		//case *event.KeyUp:
		// TODO
	}
	return nil, false
}
func (tard *TextAreaReader) kdToBytes(ev *event.KeyDown) ([]byte, bool) {

	esc := func(s string) []byte {
		mods := keyMods(ev.Mods)
		s2 := tard.escSeq(mods + s)
		return []byte(s2)
	}

	ctrl := func(b byte) byte {
		if b == '?' { // Ctrl+? → DEL
			return 0x7F
		}
		return b & 0x1F
	}

	switch ev.KeySym {
	case event.KSymReturn:
		//m := tard.temu.ScrMode().LineFeedNewlineMode()
		//if m {
		//	return []byte("\r\n"), true
		//}
		return []byte("\r"), true // vt100
	case event.KSymKeypadEnter:
		return esc("M"), true // vt100

	case event.KSymBackspace:
		m := tard.temu.ScrMode().LineFeedNewlineMode()
		if m {
			return []byte{0x7f}, true // del
		}
		return []byte{'\b'}, true

	case event.KSymUp:
		return esc("A"), true
	case event.KSymDown:
		return esc("B"), true
	case event.KSymRight:
		return esc("C"), true
	case event.KSymLeft:
		return esc("D"), true

	case event.KSymHome:
		return esc("H"), true
	case event.KSymEnd:
		return esc("F"), true

	case event.KSymEscape:
		return []byte{27}, true
	case event.KSymTab:
		return []byte{'\t'}, true

	default:

		//if ev.Mods.HasAny(event.ModShift | event.ModCtrl) {
		//return esc(string(ev.Rune)), true
		if ev.Mods.HasAny(event.ModCtrl) {
			s := string(ev.Rune)
			if len(s) == 1 {
				return []byte{ctrl(s[0])}, true
			}
		}

		return []byte(string(ev.Rune)), true
	}
}

//----------

func (tard *TextAreaReader) escSeq(seq string) string {
	appMode := tard.temu.ScrMode().CursorKeysMode()
	if appMode {
		return "\x1bO" + seq
	}
	// normal mode
	return "\x1b[" + seq
}

//----------
//----------
//----------

func keyMods(km event.KeyModifiers) string {
	mod := ""
	switch {
	case km.Is(event.ModShift):
		mod = "1;2"
	case km.Is(event.ModAlt):
		mod = "1;3"
	case km.Is(event.ModShift | event.ModAlt):
		mod = "1;4"
	case km.Is(event.ModCtrl):
		mod = "1;5"
	case km.Is(event.ModShift | event.ModCtrl):
		mod = "1;6"
	case km.Is(event.ModAlt | event.ModCtrl):
		mod = "1;7"
	case km.Is(event.ModShift | event.ModAlt | event.ModCtrl):
		mod = "1;8"
	}
	return mod
}
