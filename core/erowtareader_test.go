package core

import (
	"fmt"
	"testing"

	"github.com/jmigpin/editor/core/termemu"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/uiutil/event"
)

func TestKbEncodeToStrNewlineAndBackarrowModes(t *testing.T) {
	emu := termemu.NewEmu(nilReadWriter{}, testTui{}, termemu.Opts{})
	defer emu.Close()
	tarc := &ERowTaReadCloser{erow: &ERow{optTemu: &ERowTermEmu{emu: emu}}}

	check := func(key event.KeySym, want string) {
		t.Helper()
		if got := tarc.kbEncodeToStr(nil, &event.KeyDown{KeySym: key}); got != want {
			t.Errorf("key=%v: got %q, want %q", key, got, want)
		}
	}
	check(event.KSymReturn, "\r")
	check(event.KSymBackspace, "\x7f")

	emu.SetLNM(true)
	check(event.KSymReturn, "\r\n")
	check(event.KSymBackspace, "\x7f")

	if _, err := emu.WriteMessage([]byte("\x1b[?67h")); err != nil {
		t.Fatal(err)
	}
	check(event.KSymBackspace, "\b")
	check(event.KSymReturn, "\r\n")

	if _, err := emu.WriteMessage([]byte("\x1b[?67l")); err != nil {
		t.Fatal(err)
	}
	check(event.KSymBackspace, "\x7f")
}

func TestMouseWheelRequiresReportingMode(t *testing.T) {
	emu := termemu.NewEmu(nilReadWriter{}, testTui{}, termemu.Opts{})
	defer emu.Close()
	tarc := &ERowTaReadCloser{erow: &ERow{optTemu: &ERowTermEmu{emu: emu}, termOpts: ERowTermOpts{mouseMode: terminalMouseAuto}}}
	ev := &event.MouseDown{Button: event.ButtonWheelUp}
	check := func(want bool) {
		t.Helper()
		if got := tarc.mouseEncodeToStr(nil, ev) != ""; got != want {
			t.Errorf("mouse wheel forwarded=%v, want %v", got, want)
		}
	}
	write := func(s string) {
		t.Helper()
		if _, err := emu.WriteMessage([]byte(s)); err != nil {
			t.Fatal(err)
		}
	}
	check(false)
	write("\x1b[?1006h")
	check(false)
	tarc.erow.termOpts.mouseMode = terminalMouseOn
	check(false)
	tarc.erow.termOpts.mouseMode = terminalMouseOff
	check(false)
	tarc.erow.termOpts.mouseMode = terminalMouseAuto
	write("\x1b[?1000h")
	check(true)
	write("\x1b[?1000l")
	check(false)
}

func TestMouseInputOwnership(t *testing.T) {
	emu := termemu.NewEmu(nilReadWriter{}, testTui{}, termemu.Opts{})
	defer emu.Close()
	tarc := &ERowTaReadCloser{erow: &ERow{optTemu: &ERowTermEmu{emu: emu}, termOpts: ERowTermOpts{mouseMode: terminalMouseOn}}}
	for _, input := range []any{
		&event.MouseDown{Button: event.ButtonWheelUp},
		&event.MouseClick{Button: event.ButtonLeft},
		&event.MouseMove{},
	} {
		ev := &ui.TextAreaInputEvent{Event: input}
		tarc.onTextAreaInputEvent(ev)
		if !ev.ReplyHandled {
			t.Errorf("mouse=true did not reserve %T", input)
		}
	}
	tarc.erow.termOpts.mouseMode = terminalMouseAuto
	ev := &ui.TextAreaInputEvent{Event: &event.MouseDown{Button: event.ButtonWheelUp}}
	tarc.onTextAreaInputEvent(ev)
	if ev.ReplyHandled {
		t.Fatal("mouse=auto reserved unrequested wheel input")
	}
	checkClick := func(want bool) {
		t.Helper()
		ev := &ui.TextAreaInputEvent{Event: &event.MouseClick{Button: event.ButtonLeft}}
		tarc.onTextAreaInputEvent(ev)
		if bool(ev.ReplyHandled) != want {
			t.Errorf("mouse=auto click reserved=%v, want %v", ev.ReplyHandled, want)
		}
	}
	write := func(s string) {
		t.Helper()
		if _, err := emu.WriteMessage([]byte(s)); err != nil {
			t.Fatal(err)
		}
	}
	checkClick(false)
	write("\x1b[?1006h")
	checkClick(false)
	write("\x1b[?1000h")
	checkClick(true)
	write("\x1b[?1000l")
	checkClick(false)
}

func TestKbEncodeToStrFunctionKeys(t *testing.T) {
	keys := []struct {
		key             event.KeySym
		plain, modified string
	}{
		{event.KSymF1, "\x1bOP", "\x1b[1;%sP"},
		{event.KSymF2, "\x1bOQ", "\x1b[1;%sQ"},
		{event.KSymF3, "\x1bOR", "\x1b[1;%sR"},
		{event.KSymF4, "\x1bOS", "\x1b[1;%sS"},
		{event.KSymF5, "\x1b[15~", "\x1b[15;%s~"},
		{event.KSymF6, "\x1b[17~", "\x1b[17;%s~"},
		{event.KSymF7, "\x1b[18~", "\x1b[18;%s~"},
		{event.KSymF8, "\x1b[19~", "\x1b[19;%s~"},
		{event.KSymF9, "\x1b[20~", "\x1b[20;%s~"},
		{event.KSymF10, "\x1b[21~", "\x1b[21;%s~"},
		{event.KSymF11, "\x1b[23~", "\x1b[23;%s~"},
		{event.KSymF12, "\x1b[24~", "\x1b[24;%s~"},
	}
	modifiers := []struct {
		mods  event.KeyModifiers
		param string
	}{
		{0, ""},
		{event.ModShift, "2"},
		{event.ModAlt, "3"},
		{event.ModShift | event.ModAlt, "4"},
		{event.ModCtrl, "5"},
		{event.ModShift | event.ModCtrl, "6"},
		{event.ModAlt | event.ModCtrl, "7"},
		{event.ModShift | event.ModAlt | event.ModCtrl, "8"},
		{event.ModAltGr | event.ModCtrl | event.ModAlt, ""},
	}
	tarc := &ERowTaReadCloser{}
	for _, key := range keys {
		for _, mod := range modifiers {
			for _, locks := range []event.KeyModifiers{0, event.ModCapsLock | event.ModNumLock} {
				for _, ru := range []rune{0, 0xffbe + rune(key.key-event.KSymF1)} {
					ev := &event.KeyDown{KeySym: key.key, Mods: mod.mods | locks, Rune: ru}
					want := key.plain
					if mod.param != "" {
						want = fmt.Sprintf(key.modified, mod.param)
					}
					if got := tarc.kbEncodeToStr(nil, ev); got != want {
						t.Fatalf("key=%v mods=%v rune=%#x: got %q, want %q", key.key, ev.Mods, ru, got, want)
					}
				}
			}
		}
	}
}

func TestKbEncodeToStrAltGrRune(t *testing.T) {
	tarc := &ERowTaReadCloser{}
	ev := &event.KeyDown{
		Mods: event.ModAltGr | event.ModCtrl | event.ModAlt,
		Rune: '@',
	}

	got := tarc.kbEncodeToStr(nil, ev)
	if want := "@"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestKbEncodeToStrIgnoreAltGrModifierKey(t *testing.T) {
	tarc := &ERowTaReadCloser{}
	ev := &event.KeyDown{
		KeySym: event.KSymAltGr,
		Mods:   event.ModAltGr,
		Rune:   65027,
	}

	if got := tarc.kbEncodeToStr(nil, ev); got != "" {
		t.Fatalf("got %q, want empty string", got)
	}
}

func TestKbEncodeToStrShiftTab(t *testing.T) {
	tarc := &ERowTaReadCloser{}
	ev := &event.KeyDown{
		KeySym: event.KSymTab,
		Mods:   event.ModShift,
	}

	got := tarc.kbEncodeToStr(nil, ev)
	if want := "\x1b[Z"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	ev2 := &event.KeyDown{
		KeySym: event.KSymTabLeft,
	}
	got2 := tarc.kbEncodeToStr(nil, ev2)
	if want := "\x1b[Z"; got2 != want {
		t.Fatalf("got2 %q, want %q", got2, want)
	}
}
