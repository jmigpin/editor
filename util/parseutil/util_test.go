package parseutil

import (
	"strings"
	"testing"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func TestDetectVar(t *testing.T) {
	str := "aaaa$b $cd $e"
	if !DetectEnvVar(str, "b") {
		t.Fatal()
	}
	if !DetectEnvVar(str, "cd") {
		t.Fatal()
	}
	if !DetectEnvVar(str, "e") {
		t.Fatal()
	}

	str2 := "$a"
	if !DetectEnvVar(str2, "a") {
		t.Fatal()
	}
}

func TestAddEscapes(t *testing.T) {
	s := "a \\b"
	s2 := AddEscapes(s, '\\', " \\")
	if s2 != "a\\ \\\\b" {
		t.Fatal()
	}
	s3 := RemoveEscapes(s2, '\\')
	if s3 != s {
		t.Fatal()
	}
}

func TestExpandIndexesEscape2(t *testing.T) {
	tests := []struct {
		name  string
		src   string
		index int
		want  string
	}{
		{"word", "xx abc yy", strings.Index("xx abc yy", "b"), "abc"},
		{"escaped-space-before-index", "xx a\\ b yy", strings.Index("xx a\\ b yy", "b"), "a\\ b"},
		{"index-on-escaped-space", "xx a\\ b yy", strings.Index("xx a\\ b yy", "\\ ") + 1, "a\\ b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rd := iorw.NewStringReaderAt(tt.src)
			isNonSpace := func(ru rune) bool { return ru != ' ' }
			l, r := ExpandIndexesEscape2(rd, tt.index, false, isNonSpace, '\\')
			if got := tt.src[l:r]; got != tt.want {
				t.Fatalf("got=%q, want=%q", got, tt.want)
			}
		})
	}
}

func TestIndexLineColumn1(t *testing.T) {
	s := "123\n123\n123"
	rd := iorw.NewStringReaderAt(s)
	l, c, err := IndexLineColumn(rd, 0)
	if err != nil {
		t.Fatal(err)
	}
	i, err := LineColumnIndex(rd, l, c)
	if err != nil {
		t.Fatal(err)
	}
	if i != 0 {
		t.Fatal(i, rd.Max())
	}
}
func TestIndexLineColumn2(t *testing.T) {
	s := "123\n123\n123"
	rd := iorw.NewStringReaderAt(s)
	l, c, err := IndexLineColumn(rd, rd.Max())
	if err != nil {
		t.Fatal(err)
	}
	i, err := LineColumnIndex(rd, l, c)
	if err != nil {
		t.Fatal(err)
	}
	if i != rd.Max() {
		t.Fatal(i, rd.Max())
	}
}

func TestLineColumnIndex1(t *testing.T) {
	s := "123\n123\n123"
	rw := iorw.NewStringReaderAt(s)
	i, err := LineColumnIndex(rw, 3, 10)
	if err != nil {
		t.Fatal(err)
	}
	if i != 8 { // beginning of line
		t.Fatal(i, rw.Max())
	}
}

func TestLineColumnIndex2Bytes(t *testing.T) {
	s := "123\n123\n123"
	b := []byte(s)
	isNewline := func(ru rune) bool { return ru == '\n' }

	l, c := IndexLineColumnFn(b, 0, isNewline)
	i, err := LineColumnIndexFn(b, l, c, isNewline)
	if err != nil {
		t.Fatal(err)
	}
	if i != 0 {
		t.Fatal(i)
	}

	l2, c2 := IndexLineColumnFn(b, len(b), isNewline)
	i2, err2 := LineColumnIndexFn(b, l2, c2, isNewline)
	if err2 != nil {
		t.Fatal(err2)
	}
	if i2 != len(b) {
		t.Fatal(i2)
	}

	// test out of bounds line column behavior
	i3, err3 := LineColumnIndexFn(b, 2, 10, isNewline)
	if err3 != nil {
		t.Fatal(err3)
	}
	if i3 != 11 { // end of line 3 (index 11)
		t.Fatal(i3)
	}
}
