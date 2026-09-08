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
