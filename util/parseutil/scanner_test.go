package parseutil

import (
	"testing"
)

func TestScan1(t *testing.T) {
	s := "aa\nbbb"
	sc := newTestScanner(s)
	if sc.M.Sequence("aa") != nil {
		t.Fatal()
	}
	sc.M.ToNLExcludeOrEnd(0)
	if sc.Pos != 2 {
		t.Fatal(sc.Pos)
	}
	sc.M.NRunes(1)
	sc.M.ToNLIncludeOrEnd(0)
	if sc.Pos != len(s) {
		t.Fatal()
	}
}

func TestScan2(t *testing.T) {
	s := "file:aaab"
	sc := newTestScanner(s)

	// normal direction
	if sc.M.Sequence("file") != nil {
		t.Fatal()
	}
	if sc.Pos != 4 {
		t.Fatal(sc.Pos)
	}
	if sc.M.Sequence(":aaa") != nil {
		t.Fatal(sc.Pos)
	}

	// revert direction
	sc.Reverse = true
	if sc.M.Sequence(":aaa") != nil {
		t.Fatal(sc.Pos)
	}
	if sc.M.Sequence("file") != nil {
		t.Fatal()
	}
	if sc.Pos != 0 {
		t.Fatal(sc.Pos)
	}
}

func TestScanQuote1(t *testing.T) {
	s := `"aa\""bbb`
	es := `"aa\""`

	sc := newTestScanner(s)

	pos0 := sc.KeepPos()
	if sc.M.QuotedString() != nil {
		t.Fatal()
	}
	s2 := string(pos0.Bytes())
	if s2 != es {
		t.Fatalf("%v != %v", s, s2)
	}
}

func TestScanQuote2(t *testing.T) {
	s := `"aa`
	sc := newTestScanner(s)
	if sc.M.QuotedString() == nil {
		t.Fatal()
	}
	if sc.Pos != 0 {
		t.Fatal(sc.Pos)
	}
}

func TestEscape1(t *testing.T) {
	s := `a\`
	sc := newTestScanner(s)
	sc.Pos = 1
	if sc.M.EscapeAny('\\') == nil || sc.Pos != 1 {
		t.Fatal(sc.Pos)
	}
}
func TestEscape2(t *testing.T) {
	s := `a\\\\ `
	sc := newTestScanner(s)
	sc.Reverse = true
	sc.Pos = len(s)
	if sc.M.EscapeAny('\\') != nil || sc.Pos != 4 {
		t.Fatal(sc.Pos)
	}
}
func TestEscape3(t *testing.T) {
	s := `a\\\ `
	sc := newTestScanner(s)
	sc.Reverse = true
	sc.Pos = len(s)
	if sc.M.EscapeAny('\\') != nil || sc.Pos != 3 {
		t.Fatal(sc.Pos)
	}
}
func TestEscape4(t *testing.T) {
	s := `\\\ `
	sc := newTestScanner(s)
	sc.Reverse = true
	sc.Pos = len(s)
	if sc.M.EscapeAny('\\') != nil || sc.Pos != 2 {
		t.Fatal(sc.Pos)
	}
}

func TestInt1(t *testing.T) {
	s := `123a`
	sc := newTestScanner(s)
	pos0 := sc.KeepPos()
	if sc.M.Digits() != nil || string(pos0.Bytes()) != "123" {
		t.Fatal()
	}
}
func TestInt2(t *testing.T) {
	s := `-123`
	sc := newTestScanner(s)
	sc.Reverse = true
	sc.Pos = len(s)
	pos0 := sc.KeepPos()
	if sc.M.Integer() != nil {
		t.Fatal()
	}
	res := string(pos0.Bytes())
	if res != "-123" {
		t.Fatal(res)
	}
}

func TestFloat1(t *testing.T) {
	s := `.23`
	sc := newTestScanner(s)
	pos0 := sc.KeepPos()
	if sc.M.Float() != nil {
		t.Fatal()
	}
	res := string(pos0.Bytes())
	if res != ".23" {
		t.Fatal(res)
	}
}

//func TestFloat2(t *testing.T) {
//	s := `.23E`
//	sc := newTestScanner(s)

//	if sc.M.Float() != nil {
//		t.Fatal()
//	}
//	res := node.(float64)
//	if res != 0.23 {
//		t.Fatal(res)
//	}
//}
//func TestFloat3(t *testing.T) {
//	s := `.23E+1`
//	sc := newTestScanner(s)

//	node, err := sc.Result(sc.Float64NodeF(sc.Float))
//	if err != nil {
//		t.Fatal(err)
//	}
//	res := node.(float64)
//	if res != 2.3 {
//		t.Fatal(res)
//	}
//}
//func TestFloat4(t *testing.T) {
//	s := `00.23`
//	sc := newTestScanner(s)

//	_, err := sc.Result(sc.Float64NodeF(sc.Float))
//	if err == nil {
//		t.Fatal(err)
//	}
//}
//func TestFloat5(t *testing.T) {
//	s := `12.23`
//	sc := newTestScanner(s)

//	node, err := sc.Result(sc.Float64NodeF(sc.Float))
//	if err != nil {
//		t.Fatal(err)
//	}
//	res := node.(float64)
//	if res != 12.23 {
//		t.Fatal(res)
//	}
//}

func TestSequenceExpand(t *testing.T) {
	s := `some string with a space`
	sc := newTestScanner(s)
	sc.Pos = 10

	if sc.M.RuneSequenceMid([]rune(s)) != nil || sc.Pos != len(s) {
		t.Fatal(sc.Pos)
	}

	sc.Pos = 10
	sc.Reverse = true
	if sc.M.RuneSequenceMid([]rune(s)) != nil || sc.Pos != 0 {
		t.Fatal(sc.Pos)
	}

	sc.Pos = 10
	sc.Reverse = true
	if sc.M.RuneSequenceMid([]rune("gaa")) == nil || sc.Pos != 10 {
		t.Fatal(sc.Pos)
	}
}

//----------

//func TestParse1(t *testing.T) {
//	s := `0123456789`
//	sc := newTestScanner(s)
//	fn := sc.StringNodeF(sc.AndF(
//		sc.RuneF('0'),
//		sc.OrF(
//			sc.AndF(
//				sc.RuneF('1'),
//				sc.RuneF('3'),
//			),
//			sc.AndF(
//				sc.RuneF('1'),
//				sc.RuneF('2'),
//			),
//		),
//	))
//	node, err := sc.Result(fn)
//	if err != nil {
//		t.Fatal(err)
//	}
//	res := fmt.Sprintf("%v", node)
//	if res != "012" {
//		t.Fatal(node)
//	}
//}

//----------

func newTestScanner(s string) *Scanner {
	sc := NewScanner()
	sc.SetSrc([]byte(s))
	return sc
}

//----------
//----------
//----------

//func BenchmarkScan1(b *testing.B) {
//	// usefull for cache tests if implemented

//	s := "0123456789"
//	for i := 0; i < 7; i++ {
//		s += s
//	}

//	sc := newTestScanner(s)

//	fn := sc.LoopF(
//		sc.OrF(
//			sc.RuneF('0'),
//			sc.RuneF('1'),
//			sc.RuneF('2'),
//			sc.RuneF('4'),
//			sc.RuneF('5'),
//			sc.RuneF('6'),
//			sc.RuneF('7'),
//			sc.RuneF('8'),
//			sc.RuneF('9'),
//			sc.RuneF('a'),
//			sc.RuneF('b'),
//			sc.RuneF('c'),
//			sc.RuneF('d'),
//			sc.RuneF('e'),
//			sc.NRunesF(1),
//		),
//	)

//	b.ResetTimer()
//	for i := 0; i < b.N; i++ {
//		sc.Reset()
//		_, err := sc.Result(fn)
//		if err != nil {
//			b.Fatal(err, sc.Pos, len(s))
//		}
//		if sc.Pos != len(s) {
//			b.Fatal("not at end", sc.Pos, len(s))
//		}
//	}
//}
