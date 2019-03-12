package statemach

import (
	"testing"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func TestAccept1(t *testing.T) {
	s := "//\nbbb"
	r := iorw.NewBytesReadWriter([]byte(s))
	sm := NewSM(r)
	if !sm.AcceptSequence("//") {
		t.Fatal()
	}
	sm.AcceptToNewlineOrEnd()
	if sm.Pos != 2 {
		t.Fatal()
	}
}

func TestAcceptQuote1(t *testing.T) {
	s := `"aa\""`
	r := iorw.NewBytesReadWriter([]byte(s))
	sm := NewSM(r)
	if !sm.AcceptQuoteLoop("\"", "\\") {
		t.Fatal()
	}
	s2 := sm.Value()
	if s2 != s {
		t.Fatalf("%v != %v", s, s2)
	}
}
