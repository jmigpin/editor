package core

import (
	"io"

	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/evreg"
	"github.com/jmigpin/editor/util/uiutil/event"
)

////godebug:annotatefile
////godebug:annotatefile:erow.go

type TextareaReader struct {
	handleKeybInput bool
	reg             *evreg.Regist
	pr              *io.PipeReader
	pw              *io.PipeWriter
}

func newTextareaReader(ta *ui.TextArea) *TextareaReader {
	tard := &TextareaReader{}

	tard.pr, tard.pw = io.Pipe()

	// register to handle textarea input events
	tard.reg = ta.EvReg.Add(ui.TextAreaInputEventId, tard.onTextAreaInputEvent)

	return tard
}

//----------

func (tard *TextareaReader) Read(p []byte) (int, error) {
	return tard.pr.Read(p)
}
func (tard *TextareaReader) Close() error {
	defer tard.reg.Unregister()
	_ = tard.pr.Close()
	return tard.pw.Close()
}

//----------

func (tard *TextareaReader) onTextAreaInputEvent(ev0 any) {
	ev := ev0.(*ui.TextAreaInputEvent)
	if b, ok := tard.eventToBytes(ev.Event); ok {
		ev.ReplyHandled = event.Handled(true)
		_, err := tard.pw.Write(b)
		_ = err // TODO
	}
}
func (tard *TextareaReader) eventToBytes(ev any) ([]byte, bool) {
	switch t := ev.(type) {
	case *event.KeyDown:
		if tard.handleKeybInput {
			return tard.kdToBytes(t)
		}
	}
	return nil, false
}
func (tard *TextareaReader) kdToBytes(ev *event.KeyDown) ([]byte, bool) {

	esc := func(s string) []byte {
		mod := ""
		switch {
		case ev.Mods.Is(event.ModShift):
			mod = "1;2"
		case ev.Mods.Is(event.ModAlt):
			mod = "1;3"
		case ev.Mods.Is(event.ModShift | event.ModAlt):
			mod = "1;4"
		case ev.Mods.Is(event.ModCtrl):
			mod = "1;5"
		case ev.Mods.Is(event.ModShift | event.ModCtrl):
			mod = "1;6"
		case ev.Mods.Is(event.ModAlt | event.ModCtrl):
			mod = "1;7"
		case ev.Mods.Is(event.ModShift | event.ModAlt | event.ModCtrl):
			mod = "1;8"
		}

		return []byte("\x1b[" + mod + s)
	}

	ctrl := func(b byte) byte {
		if b == '?' { // Ctrl+? → DEL
			return 0x7F
		}
		return b & 0x1F
	}

	switch ev.KeySym {
	case event.KSymReturn, event.KSymKeypadEnter:
		// TODO: might need LF behaviour? "\r\n"
		//return []byte{'\n'}, true
		return []byte{'\r'}, true

	case event.KSymUp:
		return esc("A"), true
	case event.KSymDown:
		return esc("B"), true
	case event.KSymRight:
		return esc("C"), true
	case event.KSymLeft:
		return esc("D"), true

	case event.KSymEscape:
		return []byte{27}, true
	case event.KSymTab:
		return []byte{'\t'}, true
	case event.KSymBackspace:
		return []byte{'\b'}, true

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
