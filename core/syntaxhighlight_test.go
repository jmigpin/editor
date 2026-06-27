package core

import (
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
