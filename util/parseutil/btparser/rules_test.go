package btparser

import (
	"slices"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
)

func TestParse1(t *testing.T) {
	src := "a1  a2  \ta3a2a1"

	g := NewRules()
	ps := NewParserStateFromString(src)

	ps.Ignore = g.Spaces()

	_, err := g.Parse(ps, g.And(
		g.Loop1(g.Token(g.Or(
			g.Seq("a1"),
			g.Seq("a2"),
			g.Seq("a3"),
			g.Seq("a3a2"),
		))),
		g.Eof(),
	))
	if err != nil {
		t.Fatal(err)
	}
}

func TestParse2(t *testing.T) {
	src := "fn1(  a1=123.45 , a2=4,) "

	g := NewRules()
	ps := NewParserStateFromString(src)

	ps.Ignore = g.Spaces()

	_, err := g.Parse(ps, g.And(
		g.Token(g.Seq("fn1")),
		g.Token(g.Seq("(")),
		g.LoopSep(true,
			g.And(
				g.Token(g.Identifier()),
				g.Token(g.Seq("=")),
				g.Token(g.Or(
					g.Float(),
					g.Integer(),
				)),
			),
			g.Token(g.Seq(",")),
		),
		g.Token(g.Seq(")")),
		g.Token(g.Eof()),
	))
	if err != nil {
		t.Fatal(err)
	}
}

func TestInteger1(t *testing.T) {
	src := "0 123 -15 0 +9999"

	g := NewRules()
	ps := NewParserStateFromString(src)
	ps.Ignore = g.Spaces()

	w := []int{}

	_, err := g.Parse(ps, g.And(
		g.Loop1(
			g.Token(AppendLocal(&w, g.VInteger())),
		),
		g.Eof(),
	))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(w, []int{0, 123, -15, 0, 9999}) {
		t.Fatal(w)
	}
}

func TestRulesNamespace1(t *testing.T) {
	src := "0 123 -15 0 +9999"

	g := NewRules()
	ps := NewParserStateFromString(src)
	ps.Ignore = g.Spaces()

	w := []int{}

	_, err := g.Parse(ps, g.And(
		g.Loop1(
			g.Token(AppendLocal(&w, g.VInteger())),
		),
		g.Eof(),
	))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(w, []int{0, 123, -15, 0, 9999}) {
		t.Fatal(w)
	}
}

func TestInteger2(t *testing.T) {
	src := "+0 01 +-3 09"

	g := NewRules()
	ps := NewParserStateFromString(src)
	ps.Ignore = g.Spaces()

	w := []any{}

	_, err := g.Parse(ps, g.And(
		g.Loop1(g.And(
			g.Token(AppendLocal(&w, VOr(
				VAny(g.VInteger()),
				VAny(g.VString(g.AnyRune())),
			))),
		)),
		g.Eof(),
	))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(w, []any{"+", 0, "0", 1, "+", -3, "0", 9}) {
		t.Fatal(w)
	}
}

func TestFloat1(t *testing.T) {
	src := "-0.38 -10.65"

	g := NewRules()
	ps := NewParserStateFromString(src)
	ps.Ignore = g.Spaces()

	w := []any{}

	_, err := g.Parse(ps, g.And(
		g.Loop1(g.And(
			g.Token(AppendLocal(&w, VOr(
				VAny(g.VFloat()),
				VAny(g.VBytes(g.AnyRune())),
			))),
		)),
		g.Eof(),
	))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(w, []any{-0.38, -10.65}) {
		t.Fatal(w)
	}
}

func TestEscape1(t *testing.T) {
	src := `a\\\\`

	g := NewRules()
	ps := NewParserStateFromString(src)

	w := []any{}

	_, err := g.Parse(ps, g.And(
		g.Loop1(g.And(
			g.Token(AppendLocal(&w,
				VOr(
					VAny(g.VString(g.Escape('\\'))),
					VAny(g.VString(g.AnyRune())),
				)),
			),
		)),
		g.Eof(),
	))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(w, []any{"a", "\\\\", "\\\\"}) {
		t.Fatal(w)
	}
}

func TestValues1(t *testing.T) {
	src := "a1=1   a2=true   a3=3.4   a4=\"bcd\""

	g := NewRules()
	ps := NewParserStateFromString(src)
	ps.Ignore = g.Spaces()

	type Data struct {
		id  string
		val any
	}
	w := []*Data{}

	vDataFn := func(ps *ParserState, pos Pos) (*Data, MPos, error) {
		v := Data{}
		mp, err := g.And(
			g.Token(AssignLocal(&v.id, g.VIdentifier())),
			g.Token(g.Seq("=")),
			g.Token(AssignLocal(&v.val, VOr(
				VAny(g.VFloat()),
				VAny(g.VInteger()),
				VAny(g.VBool()),
				VAny(g.VString(g.QuotedString1())),
			))),
		)(ps, pos)
		return &v, mp, err
	}

	_, err := g.Parse(ps, g.And(
		g.Loop1(AppendLocal(&w, vDataFn)),
		g.Eof(),
	))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.EqualFunc(w, []*Data{
		&Data{id: "a1", val: 1},
		&Data{id: "a2", val: true},
		&Data{id: "a3", val: 3.4},
		&Data{id: "a4", val: "\"bcd\""},
	}, func(a, b *Data) bool {
		return a.id == b.id && a.val == b.val
	}) {
		t.Fatal(spew.Sdump(w))
	}
}

func TestRulesSemanticValueNames(t *testing.T) {
	src := "a1=1 a2=true a3=3.4 a4=\"bcd\""

	g := NewRules()
	ps := NewParserStateFromString(src)
	ps.Ignore = g.Spaces()

	type Data struct {
		id  string
		val any
	}
	w := []*Data{}

	vDataFn := func(ps *ParserState, pos Pos) (*Data, MPos, error) {
		v := Data{}
		mp, err := g.And(
			g.Token(AssignLocal(&v.id, g.VIdentifier())),
			g.Token(g.Seq("=")),
			g.Token(AssignLocal(&v.val, VOr(
				VAny(g.VFloat()),
				VAny(g.VInteger()),
				VAny(g.VBool()),
				VAny(g.VQuotedString1()),
			))),
		)(ps, pos)
		return &v, mp, err
	}

	_, err := g.Parse(ps, g.And(
		g.Loop1(AppendLocal(&w, vDataFn)),
		g.Eof(),
	))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.EqualFunc(w, []*Data{
		{id: "a1", val: 1},
		{id: "a2", val: true},
		{id: "a3", val: 3.4},
		{id: "a4", val: "bcd"},
	}, func(a, b *Data) bool {
		return a.id == b.id && a.val == b.val
	}) {
		t.Fatal(spew.Sdump(w))
	}
}

func TestEmptyLinesWithComments(t *testing.T) {
	src := "\n\n\t//C\n\ta\n   \n//C\n//C  \n  \n\n\nb//C  \n\nc\n"

	g := NewRules()
	ps := NewParserStateFromString(src)

	comments := func(ps *ParserState, pos Pos) (MPos, error) {
		return g.And(
			g.Seq("//"),
			g.LoopToNLOrEof(0, false),
		)(ps, pos)
	}

	ps.Ignore = g.EmptyLinesExceptNewline(g.Or(
		g.SpacesExceptNewline(),
		comments,
	))

	w := []string{}
	_, err := g.Parse(ps, g.And(
		g.Token(AppendLocal(&w, g.VString(g.Rune('a')))),
		g.Token(g.Newline()),
		g.Token(AppendLocal(&w, g.VString(g.Rune('b')))),
		g.Token(g.Newline()),
		g.Token(AppendLocal(&w, g.VString(g.Rune('c')))),
		g.Token(g.Newline()),
		g.Token(g.Eof()),
	))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(w, []string{"a", "b", "c"}) {
		t.Fatal(spew.Sdump(w))
	}
}

func TestLoopToNLOrEofAtEOF(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		includeNL bool
		want      string
		wantErr   bool
	}{
		{"empty_exclude_nl", "", false, "", false},
		{"empty_include_nl", "", true, "", false},
		{"line_exclude_nl", "abc", false, "abc", false},
		{"line_include_nl", "abc", true, "abc", false},
		{"newline_exclude_nl", "\n", false, "", true},
		{"newline_include_nl", "\n", true, "\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewRules()
			ps := NewParserStateFromString(tt.src)

			got := ""
			_, err := g.Parse(ps, g.And(
				AssignLocal(&got, g.VString(g.LoopToNLOrEof(0, tt.includeNL))),
				g.Eof(),
			))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoop1NoProgress(t *testing.T) {
	g := NewRules()
	ps := NewParserStateFromString("abc")

	_, err := g.Parse(ps, g.Loop1(g.NoOp()))
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsFatalError(err) {
		t.Fatalf("expected fatal error: %v", err)
	}
	if err.Error() == "" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoopSepNoProgress(t *testing.T) {
	t.Run("sep", func(t *testing.T) {
		g := NewRules()
		ps := NewParserStateFromString("abc")

		_, err := g.Parse(ps, g.LoopSep(true, g.AnyRune(), g.NoOp()))
		if err == nil {
			t.Fatal("expected error")
		}
		if !IsFatalError(err) {
			t.Fatalf("expected fatal error: %v", err)
		}
	})

	t.Run("elem", func(t *testing.T) {
		g := NewRules()
		ps := NewParserStateFromString("abc")

		_, err := g.Parse(ps, g.LoopSep(true, g.NoOp(), g.Rune(',')))
		if err == nil {
			t.Fatal("expected error")
		}
		if !IsFatalError(err) {
			t.Fatalf("expected fatal error: %v", err)
		}
	})

	t.Run("empty-elem", func(t *testing.T) {
		g := NewRules()
		ps := NewParserStateFromString(",,")

		p2, err := g.Parse(ps, g.And(
			g.LoopSepAllowEmpty(g.NoOp(), g.Rune(',')),
			g.Eof(),
		))
		if err != nil {
			t.Fatal(err)
		}
		if p2 != 2 {
			t.Fatalf("got=%v, want=%v", p2, 2)
		}
	})
}

func TestSop(t *testing.T) {
	g := NewRules()
	ps := NewParserStateFromString("abc")

	p2, err := g.ParseAt(ps, 1, g.And(
		g.Sop(),
		g.Rune('b'),
	))
	if err != nil {
		t.Fatal(err)
	}
	if p2 != 2 {
		t.Fatalf("got=%v, want=%v", p2, 2)
	}

	_, err = g.ParseAt(ps, 1, g.And(
		g.Rune('b'),
		g.Sop(),
	))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReverseSource(t *testing.T) {
	g := NewRules()
	ps := NewParserStateFromString("ab界de")
	pos := Pos(len("ab界"))

	p2, err := g.ParseAt(ps, pos, g.ReverseSource(g.Seq("界ba")))
	if err != nil {
		t.Fatal(err)
	}
	if p2 != 0 {
		t.Fatalf("got=%v, want=%v", p2, 0)
	}

	mp, err := g.ReverseSource(g.Seq("界ba"))(ps, pos)
	if err != nil {
		t.Fatal(err)
	}
	if got := ps.SourceStr(mp); got != "ab界" {
		t.Fatalf("got=%q, want=%q", got, "ab界")
	}

	ps = NewParserStateFromString("界xçde")
	pos = Pos(len("界xç"))
	mp, err = g.ReverseSource(g.Seq("çx"))(ps, pos)
	if err != nil {
		t.Fatal(err)
	}
	if got := ps.SourceStr(mp); got != "xç" {
		t.Fatalf("got=%q, want=%q", got, "xç")
	}

	ps = NewParserStateFromString("ab界de")
	pos = Pos(len("ab"))
	mp, err = g.WithBounds(2, 0, g.ReverseSource(g.Seq("ba")))(ps, pos)
	if err != nil {
		t.Fatal(err)
	}
	if got := ps.SourceStr(mp); got != "ab" {
		t.Fatalf("got=%q, want=%q", got, "ab")
	}

	ps = NewParserStateFromString("abcd")
	pos = Pos(len("ab"))
	mp, err = g.WithBounds(2, 2, g.ReverseSource(g.And(
		g.PeekBackN(2, g.Seq("dc")),
		g.Seq("ba"),
	)))(ps, pos)
	if err != nil {
		t.Fatal(err)
	}
	if got := ps.SourceStr(mp); got != "ab" {
		t.Fatalf("got=%q, want=%q", got, "ab")
	}
}

func TestWithLineBounds(t *testing.T) {
	g := NewRules()
	ps := NewParserStateFromString("aa\nbb\ncc\ndd")
	pos := Pos(len("aa\nbb\n"))

	mp, err := g.WithLineBounds(1, 1, g.Seq("cc"))(ps, pos)
	if err != nil {
		t.Fatal(err)
	}
	if got := ps.SourceStr(mp); got != "cc" {
		t.Fatalf("got=%q, want=%q", got, "cc")
	}

	mp, err = g.WithLineBounds(1, 1, g.ReverseSource(g.And(
		g.PeekBackN(len("cc"), g.Seq("cc")),
		g.Seq("\nbb"),
	)))(ps, pos)
	if err != nil {
		t.Fatal(err)
	}
	if got := ps.SourceStr(mp); got != "bb\n" {
		t.Fatalf("got=%q, want=%q", got, "bb\\n")
	}
}

func TestWithBounds(t *testing.T) {
	g := NewRules()
	ps := NewParserStateFromString("a界bc")

	p2, err := g.Parse(ps, g.WithBounds(0, len("a界"), g.Seq("a界")))
	if err != nil {
		t.Fatal(err)
	}
	if p2 != Pos(len("a界")) {
		t.Fatalf("got=%v, want=%v", p2, len("a界"))
	}

	_, err = g.Parse(ps, g.WithBounds(0, len("a界"), g.Seq("a界b")))
	if err == nil {
		t.Fatal("expected error")
	}

	ps = NewParserStateFromString("ab界cd")
	pos := Pos(len("a"))
	mp, err := g.WithBounds(1, len("b界c"), g.Seq("b界c"))(ps, pos)
	if err != nil {
		t.Fatal(err)
	}
	if got := ps.SourceStr(mp); got != "b界c" {
		t.Fatalf("got=%q, want=%q", got, "b界c")
	}

	ps = NewParserStateFromString("aa/bb/cc")
	pos = Pos(len("aa/bb"))
	mp, err = g.WithBounds(-1, -1, g.ToLastIndexByte('/'))(ps, pos)
	if err != nil {
		t.Fatal(err)
	}
	if mp.End != Pos(len("aa")) {
		t.Fatalf("got=%v, want=%v", mp.End, len("aa"))
	}

	ps = NewParserStateFromString("aa/bb/cc")
	pos = Pos(len("aa/"))
	mp, err = g.WithBounds(0, len("bb"), func(ps *ParserState, pos Pos) (MPos, error) {
		return MPos{Start: pos, End: pos + Pos(len("bb"))}, nil
	})(ps, pos)
	if err != nil {
		t.Fatal(err)
	}
	if mp.Start != Pos(len("aa/")) || mp.End != Pos(len("aa/bb")) {
		t.Fatalf("got=%v, want=%v", mp, MPos{Start: Pos(len("aa/")), End: Pos(len("aa/bb"))})
	}
}

func TestToIndexByte(t *testing.T) {
	g := NewRules()
	ps := NewParserStateFromString("aa\nbb\ncc")

	pos := Pos(len("aa\nbb"))
	mp, err := g.ToLastIndexByte('\n')(ps, pos)
	if err != nil {
		t.Fatal(err)
	}
	if mp.End != Pos(len("aa")) {
		t.Fatalf("got=%v, want=%v", mp.End, len("aa"))
	}

	mp, err = g.ToLastIndexByteOrStart('\n')(ps, pos)
	if err != nil {
		t.Fatal(err)
	}
	if got := ps.SourceStr(mp); got != "\nbb" {
		t.Fatalf("got=%q, want=%q", got, "\nbb")
	}
	if mp.End != Pos(len("aa")) {
		t.Fatalf("got=%v, want=%v", mp.End, len("aa"))
	}

	mp, err = g.ToIndexByte('\n')(ps, Pos(len("aa\n")))
	if err != nil {
		t.Fatal(err)
	}
	if got := ps.SourceStr(mp); got != "bb" {
		t.Fatalf("got=%q, want=%q", got, "bb")
	}
	if mp.End != Pos(len("aa\nbb")) {
		t.Fatalf("got=%v, want=%v", mp.End, len("aa\nbb"))
	}

	mp, err = g.ToLastIndexByteOrStart('\n')(ps, Pos(len("aa")))
	if err != nil {
		t.Fatal(err)
	}
	if mp.End != 0 {
		t.Fatalf("got=%v, want=%v", mp.End, 0)
	}

	_, err = g.ToLastIndexByte('\n')(ps, Pos(len("aa")))
	if err == nil {
		t.Fatal("expected error")
	}

	_, err = g.ToIndexByte('\n')(ps, Pos(len("aa\nbb\n")))
	if err == nil {
		t.Fatal("expected error")
	}

	mp, err = g.ToIndexByteOrEnd('\n')(ps, Pos(len("aa\nbb\n")))
	if err != nil {
		t.Fatal(err)
	}
	if mp.End != Pos(len("aa\nbb\ncc")) {
		t.Fatalf("got=%v, want=%v", mp.End, len("aa\nbb\ncc"))
	}
}

func TestIsTrue(t *testing.T) {
	g := NewRules()
	ps := NewParserStateFromString("abc")

	p2, err := g.Parse(ps, g.And(
		g.IsTrue(true),
		g.Rune('a'),
	))
	if err != nil {
		t.Fatal(err)
	}
	if p2 != 1 {
		t.Fatalf("got=%v, want=%v", p2, 1)
	}

	_, err = g.Parse(ps, g.And(
		g.IsTrue(false),
		g.Rune('a'),
	))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAssignFnDstOnlyOnSuccess(t *testing.T) {
	g := NewRules()
	ps := NewParserStateFromString("a")

	called := false
	v := ""
	_, err := g.Parse(ps, AssignFn(
		func(ps *ParserState) *string {
			called = true
			return &v
		},
		g.VString(g.Rune('b')),
	))
	if err == nil {
		t.Fatal("expected error")
	}
	if called {
		t.Fatal("dst called on failed match")
	}
}

func TestSetMapEntryFnDstOnlyOnSuccess(t *testing.T) {
	g := NewRules()
	ps := NewParserStateFromString("a")

	called := false
	m := map[string]int{}
	_, err := g.Parse(ps, SetMapEntryFn(
		func(ps *ParserState) *map[string]int {
			called = true
			return &m
		},
		VConst(g.Rune('b'), MapEntry[string, int]{Key: "k", Value: 1}),
	))
	if err == nil {
		t.Fatal("expected error")
	}
	if called {
		t.Fatal("dst called on failed match")
	}
}

func TestLookback(t *testing.T) {
	src := "--ab0--cd0--"

	g := NewRules()
	ps := NewParserStateFromString(src)

	str := ""
	strPos := Pos(0)
	_, err := g.Parse(ps, g.And(
		g.Loop1(g.Or(
			AssignLocal(&str, g.VString(
				g.DebugAnd(false, "back",
					g.And(
						g.Rune('0'),
						g.PeekBackN(2+1, g.Seq("cd")),
						func(ps *ParserState, pos Pos) (MPos, error) {
							strPos = pos - 1
							return MPos{pos, pos}, nil
						},
					),
				),
			)),
			g.AnyRune(),
		)),
		g.Eof(),
	))
	if err != nil {
		t.Fatal(err)
	}
	if str != "0" || strPos != 9 {
		t.Fatal(str, strPos)
	}
}

func TestTime(t *testing.T) {
	src := "  2025/04/02  "

	g := NewRules()
	ps := NewParserStateFromString(src)

	ps.Ignore = g.Spaces()

	date := time.Time{}
	_, err := g.Parse(ps, g.And(
		g.Token(AssignLocal(&date, g.VTime("2006/01/02"))),
		g.Token(g.Eof()),
	))
	if err != nil {
		t.Fatal(err)
	}
	if date.Month() != 4 {
		t.Fatal(date)
	}
}

func TestQuotedString(t *testing.T) {
	tests := []struct {
		in     string
		want   string
		hasErr bool
	}{
		{`"hello"`, "hello", false},
		{`"line\nbreak"`, "line\nbreak", false},
		{`"quote: \""`, `quote: "`, false},
		{`"tab\tchar"`, "tab\tchar", false},
		{`"unicode: \u263A"`, "unicode: ☺", false},
		{`"aa\"bb"`, "aa\"bb", false},
		{`"bad\xescape"`, "", true},
		{`notquoted`, "", true},
	}

	parseAndUnquote := func(s string) (string, error) {
		g := NewRules()
		ps := NewParserStateFromString(s)
		v := ""
		_, err := g.Parse(ps, g.And(
			AssignLocal(&v, g.VQuotedString1()),
			g.Eof(),
		))
		return v, err
	}

	for _, tt := range tests {
		got, err := parseAndUnquote(tt.in)
		if (err != nil) != tt.hasErr || got != tt.want {
			t.Errorf("in: %q, got: %q, want: %q, err: %v", tt.in, got, tt.want, err)
		}
	}
}

//----------

//func TestLookLimit(t *testing.T) {
//	src := "0123456789"

//	p := NewParser()
//	ps.SetSrc(src)

//	str := ""
//	_, err := g.Parse(ps, g.And(
//		g.Loop1(...),
//		g.Eof(),
//	))
//	if err != nil {
//		t.Fatal(err)
//	}
//	if str != "0" || strPos != 9 {
//		t.Fatal(str, strPos)
//	}
//}

//----------
//----------
//----------

func BenchmarkParse1(b *testing.B) {
	s := "0123456789"
	for i := 0; i < 10; i++ {
		s += s
	}

	g := NewRules()

	fn := g.Loop1(g.Or(
		g.Seq("0"),
		g.Seq("1"),
		g.Seq("2"),
		//g.Seq("3"), // commented: force accepting at the end
		g.Seq("4"),
		g.Seq("5"),
		g.Seq("6"),
		g.Seq("7"),
		g.Seq("8"),
		//g.Seq("9"),
		g.Seq("a"),
		g.Seq("b"),
		g.Seq("c"),
		g.Seq("d"),
		g.Seq("e"),
		g.Seq("f"),

		g.Seq("a12345"),
		g.Seq("b12345"),
		g.Seq("c12345"),
		g.Seq("d12345"),

		g.AnyRune(),
	))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps := NewParserStateFromString(s)
		p2, err := g.Parse(ps, fn)
		if err != nil {
			b.Fatal(err)
		}
		if p2 != Pos(len(s)) {
			b.Fatal(p2)
		}
	}
}

func BenchmarkParse1Bytes(b *testing.B) {
	s := "0123456789"
	for i := 0; i < 10; i++ {
		s += s
	}
	src := []byte(s)

	g := NewRules()

	fn := g.Loop1(g.Or(
		g.Seq("0"),
		g.Seq("1"),
		g.Seq("2"),
		//g.Seq("3"), // commented: force accepting at the end
		g.Seq("4"),
		g.Seq("5"),
		g.Seq("6"),
		g.Seq("7"),
		g.Seq("8"),
		//g.Seq("9"),
		g.Seq("a"),
		g.Seq("b"),
		g.Seq("c"),
		g.Seq("d"),
		g.Seq("e"),
		g.Seq("f"),

		g.Seq("a12345"),
		g.Seq("b12345"),
		g.Seq("c12345"),
		g.Seq("d12345"),

		g.AnyRune(),
	))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps := NewParserStateFromBytes(src)
		p2, err := g.Parse(ps, fn)
		if err != nil {
			b.Fatal(err)
		}
		if p2 != Pos(len(src)) {
			b.Fatal(p2)
		}
	}
}

func TestSeqOrMid(t *testing.T) {
	g := NewRules()
	ps := NewParserStateFromString("a.txt, line 10")

	p2, err := g.ParseAt(ps, Pos(len("a.txt, li")), g.SeqOrMid(", line "))
	if err != nil {
		t.Fatal(err)
	}
	if p2 != Pos(len("a.txt, line ")) {
		t.Fatalf("got=%v, want=%v", p2, len("a.txt, line "))
	}

	ps = NewParserStateFromString("abc")
	p2, err = g.ParseAt(ps, Pos(1), g.SeqOrMid("abc"))
	if err != nil {
		t.Fatal(err)
	}
	if p2 != Pos(len("abc")) {
		t.Fatalf("got=%v, want=%v", p2, len("abc"))
	}

	ps = NewParserStateFromString("abc")
	_, err = g.ParseAt(ps, Pos(len("abc")), g.SeqOrMid("abc"))
	if err == nil {
		t.Fatal("expected no match")
	}

	ps = NewParserStateFromString("abc xxxxx")
	_, err = g.ParseAt(ps, Pos(len("abc xxxxx")), g.SeqOrMid("abc"))
	if err == nil {
		t.Fatal("expected no match")
	}
}

func TestLastAnyRune(t *testing.T) {
	g := NewRules()
	ps := NewParserStateFromString("a界")

	mp, err := g.LastAnyRune()(ps, Pos(len("a界")))
	if err != nil {
		t.Fatal(err)
	}
	if mp.Start != Pos(len("a界")) {
		t.Fatalf("got=%v, want=%v", mp.Start, len("a界"))
	}
	if mp.End != Pos(len("a")) {
		t.Fatalf("got=%v, want=%v", mp.End, len("a"))
	}
	if got := ps.SourceStr(mp); got != "界" {
		t.Fatalf("got=%q, want=%q", got, "界")
	}
}

func TestRuneAnyOf(t *testing.T) {
	g := NewRules()
	ps := NewParserStateFromString("/")

	p2, err := g.Parse(ps, g.And(
		g.RuneAnyOf('/', '/', 0),
		g.Eof(),
	))
	if err != nil {
		t.Fatal(err)
	}
	if p2 != 1 {
		t.Fatalf("got=%v, want=%v", p2, 1)
	}
}
