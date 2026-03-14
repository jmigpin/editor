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
