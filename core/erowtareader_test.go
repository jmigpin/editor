package core

import (
	"testing"

	"github.com/jmigpin/editor/util/uiutil/event"
)

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
