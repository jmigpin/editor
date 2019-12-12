package scanutil

import (
	"testing"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func TestScan1(t *testing.T) {
	s := "//\nbbb"
	r := iorw.NewStringReader(s)
	sc := NewScanner(r)
	if !sc.Match.Sequence("//") {
		t.Fatal()
	}
	sc.Match.ToNewlineOrEnd()
	if sc.Pos != 2 {
		t.Fatal()
	}
	sc.Match.NRunes(1)
	sc.Match.ToNewlineOrEnd()
	if sc.Pos != len(s) {
		t.Fatal()
	}
}

func TestScan2(t *testing.T) {
	s := "file:aaab"
	r := iorw.NewStringReader(s)
	sc := NewScanner(r)

	// normal direction
	if !sc.Match.Sequence("file") {
		t.Fatal()
	}
	if sc.Pos != 4 {
		t.Fatal(sc.Pos)
	}
	if !sc.Match.Sequence(":aaa") {
		t.Fatal()
	}

	// revert direction
	sc.Reverse = true
	if !sc.Match.Sequence(":aaa") {
		t.Fatal()
	}
	if !sc.Match.Sequence("file") {
		t.Fatal()
	}
	if sc.Pos != 0 {
		t.Fatal(sc.Pos)
	}
}

func TestScanQuote1(t *testing.T) {
	s := `"aa\""bbb`
	es := `"aa\""`

	r := iorw.NewStringReader(s)

	sc := NewScanner(r)
	if !sc.Match.Quote('"', '\\', false, 1000) {
		t.Fatal()
	}
	s2 := sc.Value()
	if s2 != es {
		t.Fatalf("%v != %v", s, s2)
	}

	// reset for 2nd test
	sc.Start = 0
	sc.Pos = 0
	if !sc.Match.Quote('"', '\\', false, 1000) {
		t.Fatal()
	}
	s3 := sc.Value()
	if s3 != es {
		t.Fatalf("%v != %v", s, s3)
	}
}

func TestScanQuote2(t *testing.T) {
	s := `"aa`
	r := iorw.NewStringReader(s)
	sc := NewScanner(r)
	if sc.Match.Quote('"', '\\', false, 1000) {
		t.Fatal()
	}
	if sc.Pos != 0 {
		t.Fatal(sc.Pos)
	}
}

func TestEscape1(t *testing.T) {
	s := `a\`
	r := iorw.NewStringReader(s)
	sc := NewScanner(r)
	sc.Pos = 1
	if sc.Match.Escape('\\') || sc.Pos != 1 {
		t.Fatal(sc.Pos)
	}
}
func TestEscape2(t *testing.T) {
	s := `a\\\\ `
	r := iorw.NewStringReader(s)
	sc := NewScanner(r)
	sc.Reverse = true
	sc.Pos = r.Max()
	if sc.Match.Escape('\\') || sc.Pos != 6 {
		t.Fatal(sc.Pos)
	}
}
func TestEscape3(t *testing.T) {
	s := `a\\\ `
	r := iorw.NewStringReader(s)
	sc := NewScanner(r)
	sc.Reverse = true
	sc.Pos = r.Max()
	if !sc.Match.Escape('\\') || sc.Pos != 3 {
		t.Fatal(sc.Pos)
	}
}
func TestEscape4(t *testing.T) {
	s := `\\\ `
	r := iorw.NewStringReader(s)
	sc := NewScanner(r)
	sc.Reverse = true
	sc.Pos = r.Max()
	if !sc.Match.Escape('\\') || sc.Pos != 2 {
		t.Fatal(sc.Pos)
	}
}

func TestInt1(t *testing.T) {
	s := `123a`
	r := iorw.NewStringReader(s)
	sc := NewScanner(r)
	if !sc.Match.Int() || sc.Value() != "123" {
		t.Fatal(sc.Errorf(""))
	}
}
func TestFloat1(t *testing.T) {
	s := `.23`
	r := iorw.NewStringReader(s)
	sc := NewScanner(r)
	if !sc.Match.Float() || sc.Value() != ".23" {
		t.Fatal(sc.Errorf(""))
	}
}
func TestFloat2(t *testing.T) {
	s := `.23E`
	r := iorw.NewStringReader(s)
	sc := NewScanner(r)
	if !sc.Match.Float() || sc.Value() != ".23" {
		t.Fatal(sc.Errorf(""))
	}
}
func TestFloat3(t *testing.T) {
	s := `.23E+1`
	r := iorw.NewStringReader(s)
	sc := NewScanner(r)
	if !sc.Match.Float() || sc.Value() != ".23E+1" {
		t.Fatal(sc.Errorf(""))
	}
}
