package parseutil

import (
	"fmt"
	"testing"
)

func TestScan1(t *testing.T) {
	s := "aa\nbbb"
	sc := NewScannerFromString(s)
	if !sc.Sequence("aa") {
		t.Fatal()
	}
	sc.LoopAnyToNewlineExcludeOrEnd()
	if sc.Pos != 2 {
		t.Fatal(sc.Pos)
	}
	sc.NRunes(1)
	sc.LoopAnyToNewlineIncludeOrEnd()
	if sc.Pos != len(s) {
		t.Fatal()
	}
}

func TestScan2(t *testing.T) {
	s := "file:aaab"
	sc := NewScannerFromString(s)

	// normal direction
	if !sc.Sequence("file") {
		t.Fatal()
	}
	if sc.Pos != 4 {
		t.Fatal(sc.Pos)
	}
	if !sc.Sequence(":aaa") {
		t.Fatal(sc.Pos)
	}

	// revert direction
	sc.Reverse = true
	if !sc.Sequence(":aaa") {
		t.Fatal(sc.Pos)
	}
	if !sc.Sequence("file") {
		t.Fatal()
	}
	if sc.Pos != 0 {
		t.Fatal(sc.Pos)
	}
}

func TestScanQuote1(t *testing.T) {
	s := `"aa\""bbb`
	es := `"aa\""`

	sc := NewScannerFromString(s)

	if !sc.StringNode(sc.DoubleQuotedStringF('\\')) {
		t.Fatal()
	}
	s2 := sc.PopFrontNode().(string)
	if s2 != es {
		t.Fatalf("%v != %v", s, s2)
	}
}

func TestScanQuote2(t *testing.T) {
	s := `"aa`
	sc := NewScannerFromString(s)
	if sc.DoubleQuotedString('\\') {
		t.Fatal()
	}
	if sc.Pos != 0 {
		t.Fatal(sc.Pos)
	}
}

func TestEscape1(t *testing.T) {
	s := `a\`
	sc := NewScannerFromString(s)
	sc.Pos = 1
	if sc.EscapedRune('\\') || sc.Pos != 1 {
		t.Fatal(sc.Pos)
	}
}
func TestEscape2(t *testing.T) {
	s := `a\\\\ `
	sc := NewScannerFromString(s)
	sc.Reverse = true
	sc.Pos = len(s)
	if sc.EscapedRune('\\') || sc.Pos != 6 {
		t.Fatal(sc.Pos)
	}
}
func TestEscape3(t *testing.T) {
	s := `a\\\ `
	sc := NewScannerFromString(s)
	sc.Reverse = true
	sc.Pos = len(s)
	if !sc.EscapedRune('\\') || sc.Pos != 3 {
		t.Fatal(sc.Pos)
	}
}
func TestEscape4(t *testing.T) {
	s := `\\\ `
	sc := NewScannerFromString(s)
	sc.Reverse = true
	sc.Pos = len(s)
	if !sc.EscapedRune('\\') || sc.Pos != 2 {
		t.Fatal(sc.Pos)
	}
}

func TestInt1(t *testing.T) {
	s := `123a`
	sc := NewScannerFromString(s)
	if !sc.StringNode(sc.Integer) || sc.PopFrontNode().(string) != "123" {
		t.Fatal()
	}
}
func TestInt2(t *testing.T) {
	s := `-123`
	sc := NewScannerFromString(s)
	sc.Reverse = true
	sc.Pos = len(s)

	node, err := sc.Result(sc.StringNodeF(sc.Integer))
	if err != nil {
		t.Fatal(err)
	}
	res := node.(string)
	if res != "-123" {
		t.Fatal(res)
	}
}

func TestFloat1(t *testing.T) {
	s := `.23`
	sc := NewScannerFromString(s)

	node, err := sc.Result(sc.Float64NodeF(sc.Float))
	if err != nil {
		t.Fatal(err)
	}
	res := node.(float64)
	if res != 0.23 {
		t.Fatal(res)
	}
}
func TestFloat2(t *testing.T) {
	s := `.23E`
	sc := NewScannerFromString(s)

	node, err := sc.Result(sc.Float64NodeF(sc.Float))
	if err != nil {
		t.Fatal(err)
	}
	res := node.(float64)
	if res != 0.23 {
		t.Fatal(res)
	}
}
func TestFloat3(t *testing.T) {
	s := `.23E+1`
	sc := NewScannerFromString(s)

	node, err := sc.Result(sc.Float64NodeF(sc.Float))
	if err != nil {
		t.Fatal(err)
	}
	res := node.(float64)
	if res != 2.3 {
		t.Fatal(res)
	}
}
func TestFloat4(t *testing.T) {
	s := `00.23`
	sc := NewScannerFromString(s)

	_, err := sc.Result(sc.Float64NodeF(sc.Float))
	if err == nil {
		t.Fatal(err)
	}
}
func TestFloat5(t *testing.T) {
	s := `12.23`
	sc := NewScannerFromString(s)

	node, err := sc.Result(sc.Float64NodeF(sc.Float))
	if err != nil {
		t.Fatal(err)
	}
	res := node.(float64)
	if res != 12.23 {
		t.Fatal(res)
	}
}

func TestSequenceExpand(t *testing.T) {
	s := `some string with a space`
	sc := NewScannerFromString(s)
	sc.Pos = 10

	ok := false

	ok = sc.SequenceExpand(s)
	if !(ok && sc.Pos == len(s)) {
		t.Fatal("test1", sc.Pos)
	}

	sc.Reset()
	sc.Pos = 10
	sc.Reverse = true
	ok = sc.SequenceExpand(s)
	if !(ok && sc.Pos == 0) {
		t.Fatal("test2", sc.Pos)
	}

	sc.Reset()
	sc.Pos = 10
	sc.Reverse = true
	ok = sc.SequenceExpand("gaa")
	if ok {
		t.Fatal("test3", sc.Pos)
	}
}

func TestSequenceExpand2(t *testing.T) {
	//s := `aaa bbb ccc ddd`
	//sc := NewScanner4d(s)
	//for i:=0;
}

//----------

func TestParse1(t *testing.T) {
	s := `0123456789`
	sc := NewScannerFromString(s)
	fn := sc.StringNodeF(sc.AndF(
		sc.RuneF('0'),
		sc.OrF(
			sc.AndF(
				sc.RuneF('1'),
				sc.RuneF('3'),
			),
			sc.AndF(
				sc.RuneF('1'),
				sc.RuneF('2'),
			),
		),
	))
	node, err := sc.Result(fn)
	if err != nil {
		t.Fatal(err)
	}
	res := fmt.Sprintf("%v", node)
	if res != "012" {
		t.Fatal(node)
	}
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

	sc := NewScannerFromString(s)

	fn := sc.LoopF(
		sc.OrF(
			sc.RuneF('0'),
			sc.RuneF('1'),
			sc.RuneF('2'),
			sc.RuneF('4'),
			sc.RuneF('5'),
			sc.RuneF('6'),
			sc.RuneF('7'),
			sc.RuneF('8'),
			sc.RuneF('9'),
			sc.RuneF('a'),
			sc.RuneF('b'),
			sc.RuneF('c'),
			sc.RuneF('d'),
			sc.RuneF('e'),
			sc.NRunesF(1),
		),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sc.Reset()
		_, err := sc.Result(fn)
		if err != nil {
			b.Fatal(err, sc.Pos, len(s))
		}
		if sc.Pos != len(s) {
			b.Fatal("not at end", sc.Pos, len(s))
		}
	}
}
