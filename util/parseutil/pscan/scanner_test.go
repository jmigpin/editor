package pscan

import "testing"

func TestScan1(t *testing.T) {
	s := "aa\nbbb\n"
	sc := newTestScanner(s)
	p := 0
	if p2, err := sc.M.And(p,
		sc.W.Sequence("aa"),
		sc.W.ToNLOrErr(false, 0),
	); err != nil {
		t.Fatal(err)
	} else if p2 != 2 {
		t.Fatal(p2)
	} else {
		p = p2
	}
	if p3, err := sc.M.And(p,
		sc.W.NRunes(1),
		sc.W.ToNLOrErr(true, 0),
	); err != nil {
		t.Fatal(err)
	} else if p3 != len(s) {
		t.Fatal(p3)
	}
}

func TestScan2(t *testing.T) {
	s := "file:aaab"
	sc := newTestScanner(s)
	p := 0

	// normal direction
	if p2, err := sc.M.Sequence(p, "file"); err != nil {
		t.Fatal()
	} else if p2 != 4 {
		t.Fatal(p2)
	} else {
		p = p2
	}
	if p2, err := sc.M.Sequence(p, ":aaa"); err != nil {
		t.Fatal(p2)
	} else {
		p = p2
	}

	// revert direction
	sc.Reverse = true
	if p2, err := sc.M.Sequence(p, ":aaa"); err != nil {
		t.Fatal(p2)
	} else {
		p = p2
	}
	if p2, err := sc.M.Sequence(p, "file"); err != nil {
		t.Fatal()
	} else {
		p = p2
	}
	if p != 0 {
		t.Fatal(p)
	}
}

func TestScanQuote1(t *testing.T) {
	s := `"aa\""bbb`
	es := `"aa\""`

	sc := newTestScanner(s)
	p := 0
	if p2, err := sc.M.QuotedString(p); err != nil {
		t.Fatal(sc.SrcError(p2, err))
	} else {
		s2 := string(sc.SrcFromTo(p, p2))
		if s2 != es {
			t.Fatalf("%v != %v", s2, es)
		}
	}
}
func TestScanQuote2(t *testing.T) {
	s := `"aa`
	sc := newTestScanner(s)
	p := 0
	if p2, err := sc.M.QuotedString(p); err != nil {
		t.Log(sc.SrcError(p2, err))
	} else {
		t.Fatal(sc.SrcSection(p2))
	}
}
func TestEscape1(t *testing.T) {
	s := `a\`
	sc := newTestScanner(s)
	p := 1
	if p2, err := sc.M.EscapeAny(p, '\\'); err != nil {
		t.Log(sc.SrcError(p2, err))
	} else {
		t.Fatal(sc.SrcSection(p2))
	}
}
func TestEscape2(t *testing.T) {
	s := `a\\\\ `
	sc := newTestScanner(s)
	p := len(s)
	sc.Reverse = true
	if p2, err := sc.M.EscapeAny(p, '\\'); err != nil || p2 != 4 {
		t.Fatal(sc.SrcError(p2, err))
	}
}
func TestEscape3(t *testing.T) {
	s := `a\\\ `
	sc := newTestScanner(s)
	p := len(s)
	sc.Reverse = true
	if p2, err := sc.M.EscapeAny(p, '\\'); err != nil || p2 != 3 {
		t.Fatal(sc.SrcError(p2, err))
	}
}
func TestEscape4(t *testing.T) {
	s := `\\\ `
	sc := newTestScanner(s)
	p := len(s)
	sc.Reverse = true
	if p2, err := sc.M.EscapeAny(p, '\\'); err != nil || p2 != 2 {
		t.Fatal(sc.SrcError(p2, err))
	}
}

//----------

func TestLoopSep0(t *testing.T) {
	s := "a,b,c,d,"
	sc := newTestScanner(s)
	p := 0

	// can have last sep
	if p2, err := testLoopSep0F1(sc, true)(p); err != nil {
		t.Fatal(sc.SrcError(p2, err))
	} else if p2 != len(s) {
		t.Fatal(sc.SrcSection(p2))
	}

	// can not have last sep
	if p2, err := testLoopSep0F1(sc, false)(p); err != nil {
		t.Fatal(sc.SrcError(p2, err))
	} else if p2 != len(s)-1 {
		t.Fatal(sc.SrcSection(p2))
	}

}
func TestLoopSep0Rev(t *testing.T) {
	s := "a,b,c,d,"
	sc := newTestScanner(s)
	p := len(s)
	sc.Reverse = true

	// can have last sep
	if p2, err := testLoopSep0F1(sc, true)(p); err != nil {
		t.Fatal(sc.SrcError(p2, err))
	} else if p2 != 0 {
		t.Fatal(sc.SrcSection(p2))
	}

	// can not have last sep
	if p2, err := testLoopSep0F1(sc, false)(p); err == nil {
		//t.Fatal(sc.SrcError(p2, err))
		t.Fatal("should not be able to parse")
	} else if p2 != len(s) {
		t.Fatal(sc.SrcSection(p2))
	}
}

func TestLoopSep0ReUse(t *testing.T) {
	s := "a,b,c,d,"
	sc := newTestScanner(s)
	p := 0

	for i := 0; i < 3; i++ {
		if p2, err := testLoopSep0F1(sc, true)(p); err != nil {
			t.Fatal(sc.SrcError(p2, err))
		} else if p2 != len(s) {
			t.Fatal(sc.SrcSection(p2))
		}
	}
}

func testLoopSep0F1(sc *Scanner, lastSep bool) MFn {
	sep := sc.W.Rune(',')
	accept := sc.W.And(
		sc.W.MustErr(sep),
		sc.M.OneRune,
	)
	return sc.W.loopSep0(
		accept,
		sep,
		lastSep,
	)
}

//----------

func TestSection1(t *testing.T) {
	s := "(a(bc)"
	sc := newTestScanner(s)
	p := 0

	f1 := testSectionF1(sc)

	if p2, err := f1(p); err != nil {
		t.Fatal(sc.SrcError(p2, err))
	} else if p2 != len(s) {
		t.Fatal(sc.SrcSection(p2))
	}
}
func TestSection2(t *testing.T) {
	s := "(ab)c)"
	sc := newTestScanner(s)
	p := len(s)
	sc.Reverse = true

	f1 := testSectionF1(sc)

	if p2, err := f1(p); err != nil {
		t.Fatal(sc.SrcError(p2, err))
	} else if p2 != 0 {
		t.Fatal(sc.SrcSection(p2))
	}
}
func testSectionF1(sc *Scanner) MFn {
	return sc.W.Section("(", ")", 0, true, 1000, false, sc.M.OneRune)
}

//----------

func TestInt1(t *testing.T) {
	s := "123a"
	sc := newTestScanner(s)
	p := 0
	if p2, err := sc.M.Digits(p); err != nil || string(sc.SrcFromTo(p, p2)) != "123" {
		t.Fatal()
	}
}
func TestInt2(t *testing.T) {
	s := "-123"
	sc := newTestScanner(s)
	p := len(s)
	sc.Reverse = true
	if p2, err := sc.M.Integer(p); err != nil || string(sc.SrcFromTo(p, p2)) != "-123" {
		t.Fatal()
	}
}

func TestFloat1(t *testing.T) {
	s := ".23"
	sc := newTestScanner(s)
	p := 0
	if p2, err := sc.M.Float(p); err != nil || string(sc.SrcFromTo(p, p2)) != ".23" {
		t.Fatal()
	}
}
func TestFloat2(t *testing.T) {
	s := `.23E`
	sc := newTestScanner(s)
	p := 0
	if v, _, err := sc.M.Float64Value(p); err != nil || v.(float64) != 0.23 {
		t.Fatal(v)
	}
}
func TestFloat3(t *testing.T) {
	s := ".23E+1"
	sc := newTestScanner(s)
	p := 0
	if v, _, err := sc.M.Float64Value(p); err != nil || v.(float64) != 2.3 {
		t.Fatal(v)
	}
}
func TestFloat4(t *testing.T) {
	s := "00.23"
	sc := newTestScanner(s)
	p := 0
	if v, _, err := sc.M.Float64Value(p); err != nil || v.(float64) != 0.23 {
		t.Fatal(err, v)
	}
}

//----------

func TestSequenceExpand(t *testing.T) {
	s := "some string with a space"
	sc := newTestScanner(s)

	p := 10
	if p2, err := sc.M.SequenceMid(p, s); err != nil || p2 != len(s) {
		t.Fatal(err, sc.SrcSection(p2))
	}

	p3 := p
	sc.Reverse = true
	if p2, err := sc.M.SequenceMid(p3, s); err != nil || p2 != 0 {
		t.Fatal(err, p2)
	}

	if p2, err := sc.M.SequenceMid(p3, "gaa"); err != nil || p2 != 10 {
		t.Log(err, sc.SrcSection(p2))
	} else {
		t.Fatal(sc.SrcSection(p2))
	}
}

//----------
//----------
//----------

func newTestScanner(s string) *Scanner {
	sc := NewScanner()
	sc.SetSrc([]byte(s))
	return sc
}

//----------
//----------
//----------

func BenchmarkScan1(b *testing.B) {
	// usefull for cache tests if implemented

	s := "0123456789"
	for i := 0; i < 7; i++ {
		s += s
	}

	sc := newTestScanner(s)

	fn := sc.W.Loop(
		sc.W.Or(
			sc.W.Rune('0'),
			sc.W.Rune('1'),
			sc.W.Rune('2'),
			sc.W.Rune('4'),
			sc.W.Rune('5'),
			sc.W.Rune('6'),
			sc.W.Rune('7'),
			sc.W.Rune('8'),
			sc.W.Rune('9'),
			sc.W.Rune('a'),
			sc.W.Rune('b'),
			sc.W.Rune('c'),
			sc.W.Rune('d'),
			sc.W.Rune('e'),
			sc.W.NRunes(1),
		),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := 0
		p2, err := fn(p)
		if err != nil {
			b.Fatal(err, p2, len(s))
		}
		if p2 != len(s) {
			b.Fatalf("not at end: %v vs %v", p2, len(s))
		}
	}
}
