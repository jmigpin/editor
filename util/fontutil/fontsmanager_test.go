package fontutil

import (
	"testing"

	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

func TestFontFaceFallbackOptionsAndCache(t *testing.T) {
	fm := NewFontsManager()
	primary := fm.mustFont(goregular.TTF, "regular")
	fallback1 := fm.mustFont(gomedium.TTF, "medium")
	fallback2 := fm.mustFont(gomono.TTF, "mono")
	fopts := NewFaceOptions(12, 72)

	faceWithoutFallbacks := primary.FontFace(fopts)
	if got := faceWithoutFallbacks.Opts.FallbackFonts(fm); len(got) != 0 {
		t.Fatalf("unexpected initial fallbacks: %v", got)
	}

	fm.AddFallbackFont(fallback1)
	faceWithGlobalFallback := primary.FontFace(fopts)
	if faceWithGlobalFallback == faceWithoutFallbacks {
		t.Fatal("face cache ignored the changed global fallbacks")
	}
	if got := faceWithGlobalFallback.Opts.FallbackFonts(fm); !sameFonts(got, []*Font{fallback1}) {
		t.Fatalf("unexpected global fallbacks: %v", got)
	}
	if got := primary.FontFace(fopts); got != faceWithGlobalFallback {
		t.Fatal("face cache did not reuse identical effective options")
	}

	fopts.SetFallbackFonts([]*Font{fallback2, fallback1, fallback2, primary})
	faceWithLocalFallbacks := primary.FontFace(fopts)
	if got := faceWithLocalFallbacks.Opts.FallbackFonts(fm); !sameFonts(got, []*Font{fallback2, fallback1}) {
		t.Fatalf("unexpected normalized local fallbacks: %v", got)
	}
	if got := primary.FontFace(fopts); got != faceWithLocalFallbacks {
		t.Fatal("face cache did not reuse normalized local fallbacks")
	}
}

func sameFonts(a, b []*Font) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
