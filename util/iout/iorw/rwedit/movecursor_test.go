package rwedit

import (
	"slices"
	"testing"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func TestJumpSymbolRuns(t *testing.T) {
	for _, tt := range []struct {
		name  string
		text  string
		start int
		left  bool
		want  []int
	}{
		{"left", "ab*c += asdf?", len("ab*c += asdf?"), true, []int{12, 8, 5, 3, 2, 0}},
		{"right", "ab*c += asdf?", 0, false, []int{2, 3, 4, 7, 12, 13}},
		{"spaces left", "abc   def", len("abc   def"), true, []int{6, 0}},
		{"spaces right", "abc   def", 0, false, []int{3, 9}},
		{"newline left", "abc\nxyz", len("abc\nxyz"), true, []int{4, 0}},
		{"newline right", "abc\nxyz", 0, false, []int{3, 7}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewCtx()
			ctx.RW = iorw.NewBytesReadWriterAt([]byte(tt.text))
			ctx.C.SetIndex(tt.start)
			got := []int{}
			for range tt.want {
				var err error
				if tt.left {
					err = MoveCursorJumpLeft(ctx, false)
				} else {
					err = MoveCursorJumpRight(ctx, false)
				}
				if err != nil {
					t.Fatal(err)
				}
				got = append(got, ctx.C.Index())
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeleteSymbolRun(t *testing.T) {
	for _, tt := range []struct {
		name  string
		start int
		f     func(*Ctx) error
		want  string
		index int
	}{
		{"backspace", 7, BackspaceWord, "ab*c  asdf?", 5},
		{"delete", 5, DeleteWord, "ab*c  asdf?", 5},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewCtx()
			ctx.RW = iorw.NewBytesReadWriterAt([]byte("ab*c += asdf?"))
			ctx.C.SetIndex(tt.start)
			if err := tt.f(ctx); err != nil {
				t.Fatal(err)
			}
			b, err := iorw.ReadFastFull(ctx.RW)
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != tt.want || ctx.C.Index() != tt.index {
				t.Fatalf("got %q at %v, want %q at %v", b, ctx.C.Index(), tt.want, tt.index)
			}
		})
	}
}
