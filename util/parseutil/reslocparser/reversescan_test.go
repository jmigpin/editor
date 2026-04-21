package reslocparser

import (
	"testing"

	"github.com/jmigpin/editor/util/testutil"
)

func TestReverseString(t *testing.T) {
	got := revStr("a€界")
	want := "界€a"
	if got != want {
		t.Fatalf("got=%q, want=%q", got, want)
	}
}

func TestReverseScanParse(t *testing.T) {
	rs := NewReverseScanResLoc('\\', '/')

	tests := []struct {
		in     string
		maxLen int
		want   string
	}{
		{in: "AAA /a/b/c●.txt BBB", maxLen: 0, want: "/a/b/c"},
		{in: "bbb 'aa.py', line● 10, in <bb>", maxLen: 0, want: "'aa.py', line"},
		{in: "bbb 'aa.py', line ●10, in <bb>", maxLen: 0, want: "'aa.py', line "},
		{in: "bbb 'aa.py', line 10●, in <bb>", maxLen: 0, want: "'aa.py', line 10"},
		{in: "/a/b.txt: line● 2: etc", maxLen: 0, want: "/a/b.txt: line"},
		{in: "/a/b.txt: line ●2: etc", maxLen: 0, want: "/a/b.txt: line "},
		{in: "/a/b.txt: line 2●: etc", maxLen: 0, want: "/a/b.txt: line 2"},
		{in: "AAA '/a/b c●.txt' BBB", maxLen: 0, want: "'/a/b c"},
		{in: "aa\nbbb /a/b/c●.txt", maxLen: 0, want: `/a/b/c`},
	}

	for _, tt := range tests {
		src, index, err := testutil.SourceCursor("●", tt.in, 0)
		if err != nil {
			t.Fatal(err)
		}

		i, err := rs.ParseStart([]byte(src), index, tt.maxLen)
		if err != nil {
			t.Fatal(err)
		}

		got := src[i:index]
		if got != tt.want {
			t.Fatalf("got=%q, want=%q", got, tt.want)
		}
	}
}

func TestReverseScanParseOnEscapeRune(t *testing.T) {
	rs := NewReverseScanResLoc('\\', '/')

	tests := []struct {
		in   string
		want string
	}{
		{in: "/a/b/c\\● c.txt", want: "/a/b/c\\"},
		{in: " a\\ b\\● c.txt", want: "a\\ b\\"},
	}

	for _, tt := range tests {
		src, index, err := testutil.SourceCursor("●", tt.in, 0)
		if err != nil {
			t.Fatal(err)
		}

		i, err := rs.ParseStart([]byte(src), index, 0)
		if err != nil {
			t.Fatal(err)
		}

		got := src[i:index]
		if got != tt.want {
			t.Fatalf("got=%q, want=%q", got, tt.want)
		}
	}
}
