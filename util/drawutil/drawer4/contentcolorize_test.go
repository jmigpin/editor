package drawer4

import (
	"image"
	"image/color"
	"testing"

	"github.com/jmigpin/editor/util/fontutil"
	"github.com/jmigpin/editor/util/iout/iorw"
)

func TestGitColorizeOps(t *testing.T) {
	addFg := color.RGBA{0, 128, 0, 255}
	delFg := color.RGBA{128, 0, 0, 255}

	d := New()
	d.SetFontFace(fontutil.FontsMan.DefaultFontFace())
	d.SetBounds(image.Rect(0, 0, 1000, 1000))
	d.SetReader(iorw.NewStringReaderAt("+a\n b\n-c\n+++ h\n--- h"))
	d.Opt.ContentColorize.Git.On = true
	d.Opt.ContentColorize.Git.AddFg = addFg
	d.Opt.ContentColorize.Git.DeleteFg = delFg

	ops := gitColorizeOps(d)
	want := []struct {
		offset int
		fg     color.Color
	}{
		{0, addFg}, {2, nil},
		{6, delFg}, {8, nil},
		{9, addFg}, {14, nil},
		{15, delFg}, {20, nil},
	}

	if len(ops) != len(want) {
		t.Fatalf("ops len: got %v, want %v", len(ops), len(want))
	}
	for i, w := range want {
		op := ops[i]
		if op.Offset != w.offset || op.Fg != w.fg {
			t.Fatalf("op[%v]: got offset=%v fg=%v, want offset=%v fg=%v", i, op.Offset, op.Fg, w.offset, w.fg)
		}
	}
}
