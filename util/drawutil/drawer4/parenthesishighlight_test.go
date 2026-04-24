package drawer4

import (
	"image/color"
	"reflect"
	"strings"
	"testing"

	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/iout/iorw"
)

func TestParenthesisHighlightSkipsString(t *testing.T) {
	s := `x("(")`
	got := parenthesisHighlightOffsets(t, s, strings.Index(s, "("))
	want := []int{1, 2, 5, 6}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%v, want=%v", got, want)
	}
}

func TestParenthesisHighlightSkipsStringReverse(t *testing.T) {
	s := `x("(")`
	got := parenthesisHighlightOffsets(t, s, strings.LastIndex(s, ")"))
	want := []int{1, 2, 5, 6}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%v, want=%v", got, want)
	}
}

func TestParenthesisHighlightInsideString(t *testing.T) {
	s := `x("(a)")`
	got := parenthesisHighlightOffsets(t, s, strings.Index(s, "(a"))
	want := []int{3, 4, 5, 6}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%v, want=%v", got, want)
	}
}

func TestParenthesisHighlightInsideStringReverse(t *testing.T) {
	s := `x("(a)")`
	got := parenthesisHighlightOffsets(t, s, strings.Index(s, ")\""))
	want := []int{3, 4, 5, 6}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%v, want=%v", got, want)
	}
}

func TestParenthesisHighlightSkipsComment(t *testing.T) {
	s := `x( /* ) */ )`
	got := parenthesisHighlightOffsets(t, s, strings.Index(s, "("), &drawutil.SyntaxComment{Start: "/*", End: "*/"})
	want := []int{1, 2, 11, 12}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%v, want=%v", got, want)
	}
}

func TestParenthesisHighlightSkipsLineComment(t *testing.T) {
	s := "x( // )\n )"
	got := parenthesisHighlightOffsets(t, s, strings.Index(s, "("), &drawutil.SyntaxComment{Start: "//"})
	want := []int{1, 2, 9, 10}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%v, want=%v", got, want)
	}
}

func parenthesisHighlightOffsets(t *testing.T, s string, cursor int, scs ...*drawutil.SyntaxComment) []int {
	t.Helper()

	d := New()
	d.SetReader(iorw.NewStringReaderAt(s))
	d.opt.cursor.offset = cursor
	d.Opt.ParenthesisHighlight.Fg = color.Black
	d.Opt.SyntaxHighlight.Comment.SCs = scs

	ph := &ParenthesisHighlight{d: d, pad: 1000}
	ops := ph.do()
	offsets := []int{}
	for _, op := range ops {
		offsets = append(offsets, op.Offset)
	}
	return offsets
}
