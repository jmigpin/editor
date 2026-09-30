package core

import (
	"fmt"
	"testing"

	"github.com/jmigpin/editor/util/uiutil/event"
)

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
