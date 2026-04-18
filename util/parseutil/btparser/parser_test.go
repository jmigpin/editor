package btparser

import (
	"slices"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
)

func TestParse1(t *testing.T) {
	src := "a1  a2  \ta3a2a1"

	p := NewParser()
	p.SetSrcFromString(src)
	g := p.G()

	p.SetIgnore(g.Spaces())

	_, err := p.Parse(g.And(
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

	p := NewParser()
	p.SetSrcFromString(src)
	g := p.G()

	p.SetIgnore(g.Spaces())

	_, err := p.Parse(g.And(
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

	p := NewParser()
	p.SetSrcFromString(src)
	g := p.G()
	p.SetIgnore(g.Spaces())

	w := []int{}

	_, err := p.Parse(g.And(
		g.Loop1(
			g.Token(Append(&w, g.VInteger())),
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

	p := NewParser()
	p.SetSrcFromString(src)
	g := p.G()
	p.SetIgnore(g.Spaces())

	w := []int{}

	_, err := p.Parse(g.And(
		g.Loop1(
			g.Token(Append(&w, g.VInteger())),
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

	p := NewParser()
	p.SetSrcFromString(src)
	g := p.G()
	p.SetIgnore(g.Spaces())

	w := []any{}

	_, err := p.Parse(g.And(
		g.Loop1(g.And(
			g.Token(Append(&w, VOr(
				VAny(g.VInteger()),
				VAny(g.VSourceStr(g.AnyRune())),
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

	p := NewParser()
	p.SetSrcFromString(src)
	g := p.G()
	p.SetIgnore(g.Spaces())

	w := []any{}

	_, err := p.Parse(g.And(
		g.Loop1(g.And(
			g.Token(Append(&w, VOr(
				VAny(g.VFloat()),
				VAny(g.VSource(g.AnyRune())),
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

	p := NewParser()
	p.SetSrcFromString(src)
	g := p.G()

	w := []any{}

	_, err := p.Parse(g.And(
		g.Loop1(g.And(
			g.Token(Append(&w,
				VOr(
					VAny(g.VSourceStr(g.Escape('\\'))),
					VAny(g.VSourceStr(g.AnyRune())),
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

	p := NewParser()
	p.SetSrcFromString(src)
	g := p.G()
	p.SetIgnore(g.Spaces())

	type Data struct {
		id  string
		val any
	}
	w := []*Data{}

	vDataFn := func(pos Pos) (*Data, MPos, error) {
		v := Data{}
		mp, err := g.And(
			g.Token(Keep(&v.id, g.VIdentifier())),
			g.Token(g.Seq("=")),
			g.Token(Keep(&v.val, VOr(
				VAny(g.VFloat()),
				VAny(g.VInteger()),
				VAny(g.VBool()),
				VAny(g.VSourceStr(g.QuotedString1())),
			))),
		)(pos)
		return &v, mp, err
	}

	_, err := p.Parse(g.And(
		g.Loop1(Append(&w, vDataFn)),
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

	p := NewParser()
	p.SetSrcFromString(src)
	g := p.G()
	p.SetIgnore(g.Spaces())

	type Data struct {
		id  string
		val any
	}
	w := []*Data{}

	vDataFn := func(pos Pos) (*Data, MPos, error) {
		v := Data{}
		mp, err := g.And(
			g.Token(Keep(&v.id, g.VIdentifier())),
			g.Token(g.Seq("=")),
			g.Token(Keep(&v.val, VOr(
				VAny(g.VFloat()),
				VAny(g.VInteger()),
				VAny(g.VBool()),
				VAny(g.VQuotedString1()),
			))),
		)(pos)
		return &v, mp, err
	}

	_, err := p.Parse(g.And(
		g.Loop1(Append(&w, vDataFn)),
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

	p := NewParser()
	p.SetSrcFromString(src)
	g := p.G()

	comments := func(pos Pos) (MPos, error) {
		return g.And(
			g.Seq("//"),
			g.LoopToNLOrEof(0, false),
		)(pos)
	}

	p.SetIgnore(g.EmptyLinesExceptNewline(g.Or(
		g.SpacesExceptNewline(),
		comments,
	)))

	w := []string{}
	_, err := p.Parse(g.And(
		g.Token(Append(&w, g.VSourceStr(g.Rune('a')))),
		g.Token(g.Newline()),
		g.Token(Append(&w, g.VSourceStr(g.Rune('b')))),
		g.Token(g.Newline()),
		g.Token(Append(&w, g.VSourceStr(g.Rune('c')))),
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

func TestLookback(t *testing.T) {
	src := "--ab0--cd0--"

	p := NewParser()
	p.SetSrcFromString(src)
	g := p.G()

	str := ""
	strPos := Pos(0)
	_, err := p.Parse(g.And(
		g.Loop1(g.Or(
			Keep(&str, g.VSourceStr(
				g.DebugAnd(false, "back",
					g.And(
						g.Rune('0'),
						g.LookbackN(2+1, g.Seq("cd")),
						func(pos Pos) (MPos, error) {
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

	p := NewParser()
	p.SetSrcFromString(src)
	g := p.G()

	p.SetIgnore(g.Spaces())

	date := time.Time{}
	_, err := p.Parse(g.And(
		g.Token(Keep(&date, g.VTime("2006/01/02"))),
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
		p := NewParser()
		p.SetSrcFromString(s)
		g := p.G()
		v := ""
		_, err := p.Parse(g.And(
			Keep(&v, g.VQuotedString1()),
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
//	p.SetSrc(src)

//	str := ""
//	_, err := p.Parse(g.And(
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
	for i := 0; i < 7; i++ {
		s += s
	}

	p := NewParser()
	g := p.G()

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
		p.SetSrcFromString(s)
		p2, err := p.Parse(fn)
		if err != nil {
			b.Fatal(err)
		}
		if p2 != 1280 {
			b.Fatal(p2)
		}
	}
}
