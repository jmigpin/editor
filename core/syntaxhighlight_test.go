package core

import (
	"image/color"
	"testing"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/iout/iorw/rwedit"
)

func TestToolbarCommentOnlyCommentsCurrentPhysicalLine(t *testing.T) {
	tests := []struct {
		name   string
		src    string
		cursor int
		want   string
	}{
		{
			name:   "first continuation line",
			src:    "Cmd a \\\n b",
			cursor: 2,
			want:   "#Cmd a \\\n b",
		},
		{
			name:   "second continuation line",
			src:    "Cmd a \\\n b",
			cursor: 10,
			want:   "Cmd a \\\n #b",
		},
		{
			name:   "part within line",
			src:    "A | Cmd b | C",
			cursor: 6,
			want:   "A | #Cmd b | C",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := rwedit.NewCtx()
			ctx.RW = iorw.NewBytesReadWriterAt([]byte(tt.src))
			ctx.C.SetIndex(tt.cursor)
			ctx.Fns.CommentLineSym = func() any { return "#" }
			ctx.Fns.CommentUnitIndexes = toolbarCommentUnitIndexes

			if err := rwedit.Comment(ctx); err != nil {
				t.Fatal(err)
			}
			gotBytes, err := iorw.ReadFastFull(ctx.RW)
			if err != nil {
				t.Fatal(err)
			}
			if got := string(gotBytes); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToolbarCommentSelectionUsesSelectedLines(t *testing.T) {
	ctx := rwedit.NewCtx()
	ctx.RW = iorw.NewBytesReadWriterAt([]byte("A | B\nC"))
	ctx.C.SetSelection(0, 7)
	ctx.Fns.CommentLineSym = func() any { return "#" }
	ctx.Fns.CommentUnitIndexes = toolbarCommentUnitIndexes

	if err := rwedit.Comment(ctx); err != nil {
		t.Fatal(err)
	}
	gotBytes, err := iorw.ReadFastFull(ctx.RW)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(gotBytes), "#A | B\n#C"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestToolbarImportantVariableSpans(t *testing.T) {
	src := "$terminal=emu | $colorize=syntax | $font=mono | $scrollMode=auto | Cmd $terminal | $other=x | # $terminal=no"
	got := toolbarImportantVariableSpans(src)
	want := [][2]int{
		{0, len("$terminal")},
		{16, 16 + len("$colorize")},
		{35, 35 + len("$font")},
		{48, 48 + len("$scrollMode")},
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
		if s := src[got[i][0]:got[i][1]]; s != src[want[i][0]:want[i][1]] {
			t.Fatalf("span %d: got %q, want %q", i, s, src[want[i][0]:want[i][1]])
		}
	}
}

func TestToolbarImportantVariableColors(t *testing.T) {
	oldFg, oldBg := toolbarVarFgColor, toolbarVarBgColor
	defer func() {
		toolbarVarFgColor, toolbarVarBgColor = oldFg, oldBg
	}()

	fg := color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0xff}
	bg := color.RGBA{R: 0x40, G: 0x50, B: 0x60, A: 0xff}

	toolbarVarFgColor = 0x008b00
	toolbarVarBgColor = 0x102030
	gotFg, gotBg := toolbarImportantVariableColors(fg, bg)
	if got, want := color.RGBAModel.Convert(gotFg), color.RGBAModel.Convert(color.RGBA{G: 0x8b, A: 0xff}); got != want {
		t.Fatalf("fg: got %v, want %v", got, want)
	}
	if got, want := color.RGBAModel.Convert(gotBg), color.RGBAModel.Convert(color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0xff}); got != want {
		t.Fatalf("bg: got %v, want %v", got, want)
	}

	toolbarVarFgColor = 1
	toolbarVarBgColor = 1
	gotFg, gotBg = toolbarImportantVariableColors(fg, bg)
	if gotFg != fg || gotBg != bg {
		t.Fatalf("disabled colors: got (%v, %v), want (%v, %v)", gotFg, gotBg, fg, bg)
	}
}
