package core

import (
	"path/filepath"
	"testing"

	"github.com/jmigpin/editor/util/fontutil"
)

func TestLoadFallbackFontsReturnsPartialResult(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.ttf")
	fonts, err := loadFallbackFonts("regular," + missing + ",mono")
	if err == nil {
		t.Fatal("expected error for missing font")
	}
	want := []*fontutil.Font{
		fontutil.FontsMan.FontByName("regular"),
		fontutil.FontsMan.FontByName("mono"),
	}
	if len(fonts) != len(want) {
		t.Fatalf("got %d loaded fonts, want %d", len(fonts), len(want))
	}
	for i := range fonts {
		if fonts[i] != want[i] {
			t.Fatalf("loaded font %d differs from expected font", i)
		}
	}
}

func TestParseTerminalMouseOpts(t *testing.T) {
	tests := []struct {
		value string
		want  terminalMouseMode
	}{
		{"", terminalMouseOff},
		{"emu", terminalMouseAuto},
		{"emu,mouse=auto", terminalMouseAuto},
		{"emu,mouse=true", terminalMouseOn},
		{"emu,mouse=false", terminalMouseOff},
		{"mouse=false,emu", terminalMouseOff},
		{"emu,no-mouse", terminalMouseOff},
		{"mouse", terminalMouseOn},
		{"no-mouse", terminalMouseOff},
	}
	erow := &ERow{}
	for _, tt := range tests {
		if got := erow.parseTerminalOpts(tt.value).mouseMode; got != tt.want {
			t.Errorf("$terminal=%q: mouseMode=%v, want %v", tt.value, got, tt.want)
		}
	}
}
