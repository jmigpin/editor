package reslocparser

import (
	"testing"

	"github.com/jmigpin/editor/util/parseutil/btparser"
	"github.com/jmigpin/editor/util/testutil"
)

func TestReverseScanParse(t *testing.T) {
	rs := NewReverseScanResLoc('\\', '/', false)

	tests := []struct {
		in     string
		maxLen int
		want   string
	}{
		{in: "AAA /a/b/c●.txt BBB", maxLen: -1, want: "/a/b/c"},
		{in: "bbb 'aa.py', line● 10, in <bb>", maxLen: -1, want: "'aa.py', line"},
		{in: "bbb 'aa.py', line ●10, in <bb>", maxLen: -1, want: "'aa.py', line "},
		{in: "bbb 'aa.py', line 10●, in <bb>", maxLen: -1, want: "'aa.py', line 10"},
		{in: "/a/b.txt: line● 2: etc", maxLen: -1, want: "/a/b.txt: line"},
		{in: "/a/b.txt: line ●2: etc", maxLen: -1, want: "/a/b.txt: line "},
		{in: "/a/b.txt: line 2●: etc", maxLen: -1, want: "/a/b.txt: line 2"},
		{in: "AAA '/a/b c●.txt' BBB", maxLen: -1, want: "'/a/b c"},
		{in: "aa\nbbb /a/b/c●.txt", maxLen: -1, want: `/a/b/c`},
	}

	for _, tt := range tests {
		src, index, err := testutil.SourceCursor("●", tt.in, 0)
		if err != nil {
			t.Fatal(err)
		}

		ps := btparser.NewParserStateFromBytes([]byte(src))
		i, err := rs.ParseStart(ps, index, tt.maxLen)
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
	rs := NewReverseScanResLoc('\\', '/', false)

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

		ps := btparser.NewParserStateFromBytes([]byte(src))
		i, err := rs.ParseStart(ps, index, -1)
		if err != nil {
			t.Fatal(err)
		}

		got := src[i:index]
		if got != tt.want {
			t.Fatalf("got=%q, want=%q", got, tt.want)
		}
	}
}

func TestReverseScanRule(t *testing.T) {
	rs := NewReverseScanResLoc('\\', '/', false)

	src, index, err := testutil.SourceCursor("●", "xx /a/b●.txt yy", 0)
	if err != nil {
		t.Fatal(err)
	}
	ps := btparser.NewParserStateFromBytes([]byte(src))
	rl := NewResLoc()
	ps.UserData[resLocDataKey] = rl

	mp, err := rs.Rule(-1)(ps, btparser.Pos(index))
	if err != nil {
		t.Fatal(err)
	}
	if mp.End != btparser.Pos(len("xx ")) {
		t.Fatalf("got=%v, want=%v", mp.End, len("xx "))
	}
}

func TestCoverIndex(t *testing.T) {
	g := btparser.NewRules()
	src := []byte("xx /a/b.txt yy")
	index := len("xx /a/b")
	ps := btparser.NewParserStateFromBytes(src)
	rl := NewResLoc()
	ps.UserData[resLocDataKey] = rl

	fn := func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
		rl := ps.UserData[resLocDataKey].(*ResLoc)
		return btparser.AssignLocal(&rl.Path, g.VString(g.Seq("/a/b.txt")))(ps, pos)
	}

	mp, err := coverIndex(index, fn)(ps, 0)
	if err != nil {
		t.Fatal(err)
	}
	if mp.Start != btparser.Pos(len("xx ")) {
		t.Fatalf("got=%v, want=%v", mp.Start, len("xx "))
	}
	if mp.End != btparser.Pos(len("xx /a/b.txt")) {
		t.Fatalf("got=%v, want=%v", mp.End, len("xx /a/b.txt"))
	}
	if rl.Path != "/a/b.txt" {
		t.Fatalf("got=%q, want=%q", rl.Path, "/a/b.txt")
	}
}
