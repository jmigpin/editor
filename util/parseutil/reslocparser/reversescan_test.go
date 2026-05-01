package reslocparser

import (
	"strings"
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

//----------
//----------
//----------

func BenchmarkReverseScanRuleVsBruteCoverPosEnd(b *testing.B) {
	tests := []struct {
		name string
		in   string
	}{
		{
			name: "shortText",
			in:   "aa bb cc /a/b/c.txt● yy",
		},
		{
			name: "longText",
			in:   strings.Repeat("alpha beta gamma delta ", 200) + "/a/b/c.txt● yy",
		},
		{
			name: "longTextMiddle",
			in:   strings.Repeat("alpha beta gamma delta ", 100) + "/a/b/c●.txt yy " + strings.Repeat("alpha beta gamma delta ", 100),
		},
	}

	for _, tt := range tests {
		src, index, err := testutil.SourceCursor("●", tt.in, 0)
		if err != nil {
			b.Fatal(err)
		}
		srcBytes := []byte(src)

		b.Run(tt.name+"/revscan", func(b *testing.B) {
			benchResLocRule(b, srcBytes, index, benchRevScanRule())
		})
		b.Run(tt.name+"/brutecoverposend", func(b *testing.B) {
			benchResLocRule(b, srcBytes, index, benchBruteCoverPosEndRule())
		})
	}
}

func benchRevScanRule() btparser.MFn {
	g := btparser.NewRules()
	rs := NewReverseScanResLoc('\\', '/', false)
	return g.And(
		rs.Rule(3000),
		benchPathRule(g),
	)
}

func benchBruteCoverPosEndRule() btparser.MFn {
	g := btparser.NewRules()
	return g.BruteCoverPosEnd(
		g.WithBounds(3000, 0,
			g.And(
				g.ToLastIndexByteOrStart('\n'),
				g.Optional(g.Rune('\n')),
			),
		),
		benchPathRule(g),
	)
}

func benchPathRule(g btparser.Rules) btparser.MFn {
	return func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
		rl := ps.UserData[resLocDataKey].(*ResLoc)
		return btparser.AssignLocal(&rl.Path, g.VString(g.Seq("/a/b/c.txt")))(ps, pos)
	}
}

func benchResLocRule(b *testing.B, src []byte, index int, fn btparser.MFn) {
	b.Helper()
	for i := 0; i < b.N; i++ {
		ps := btparser.NewParserStateFromBytes(src)
		ps.UserData[resLocDataKey] = NewResLoc()
		if _, err := fn(ps, btparser.Pos(index)); err != nil {
			b.Fatal(err)
		}
	}
}
