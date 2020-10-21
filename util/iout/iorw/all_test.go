package iorw

import (
	"bytes"
	"context"
	"testing"
	"unicode"
)

func TestRW1(t *testing.T) {
	s := "0123"
	rw := NewBytesReadWriterAt([]byte(s))
	type ow struct {
		i int
		l int
		s string
		e string // expected
	}

	var tests = []*ow{
		{1, 0, "ab", "0ab123"},
		{5, 0, "ab", "0ab12ab3"},
		{1, 2, "", "012ab3"},
		{3, 2, "", "0123"},
		{1, 0, "ab", "0ab123"},
		{0, 6, "abcde", "abcde"},
		{0, 5, "abc", "abc"},
		{0, 1, "abcd", "abcdbc"},
		{3, 2, "000", "abc000c"},
		{7, 0, "f", "abc000cf"},
	}

	for _, w := range tests {
		if err := rw.OverwriteAt(w.i, w.l, []byte(w.s)); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(rw.buf, []byte(w.e)) {
			t.Fatal(string(rw.buf) + " != " + w.e)
		}
	}
}

//----------

func TestIndex1(t *testing.T) {
	s := "0123456789"
	for i := 0; i < 32*1024; i++ {
		s += "0123456789"
	}
	s += "abc"

	rw := NewStringReaderAt(s)

	i, err := Index(rw, 4, []byte("abc"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(i)
}

func TestIndex2(t *testing.T) {
	s := "012345678"
	rw := NewStringReaderAt(s)
	i, err := indexCtx2(context.Background(), rw, 0, []byte("345"), true, 4)
	if err != nil {
		t.Fatal(err)
	}
	if i < 0 {
		t.Fatal("not found")
	}
}

func TestLastIndex1(t *testing.T) {
	s := "a\n0123\nb"
	rw := NewStringReaderAt(s)

	fn := func(ru rune) bool {
		return ru == '\n'
	}

	i, _, err := RuneLastIndexFn(rw, 6, true, fn)
	if err != nil {
		t.Fatal(err)
	}
	if i != 1 {
		t.Fatal(i)
	}
}

func TestExpandIndex1(t *testing.T) {
	s := "a 234 b"
	rw := NewStringReaderAt(s)
	i := ExpandRuneIndexFn(rw, 3, true, unicode.IsSpace)
	if i != 5 {
		t.Fatal(i)
	}
	i = ExpandRuneIndexFn(rw, i+1, true, unicode.IsSpace)
	if i != 7 {
		t.Fatal(i)
	}
}

func TestExpandLastIndex1(t *testing.T) {
	s := "a 234 b"
	rw := NewStringReaderAt(s)
	i := ExpandRuneLastIndexFn(rw, 3, true, unicode.IsSpace)
	if i != 2 {
		t.Fatal(i)
	}
	// repeat from same position
	i = ExpandRuneLastIndexFn(rw, i, true, unicode.IsSpace)
	if i != 2 {
		t.Fatal(i)
	}

	i = ExpandRuneLastIndexFn(rw, i-1, true, unicode.IsSpace)
	if i != 0 {
		t.Fatal(i)
	}
}

//----------

func TestWordAtIndex(t *testing.T) {
	s := "abc f"
	rw := NewStringReaderAt(s)
	w, i, err := WordAtIndex(rw, 3)
	if err == nil {
		t.Fatalf("%v %v %v", w, i, err)
	}
}

//----------

func TestLineStartIndex(t *testing.T) {
	s := "0123456789"
	rw := NewStringReaderAt(s)
	rw2 := NewLimitedReaderAt(rw, 3, 5)
	v, err := LineStartIndex(rw2, 4)
	if err != nil {
		t.Fatal(err)
	}
	if v != 3 {
		t.Fatal(err)
	}
}

func TestLineEndIndex(t *testing.T) {
	s := "0123456789"
	rw := NewStringReaderAt(s)
	rw2 := NewLimitedReaderAt(rw, 3, 5)
	v, newLine, err := LineEndIndex(rw2, 4)
	if err != nil {
		t.Fatal(err)
	}
	if !(v == 5 && newLine == false) {
		t.Fatal(v, newLine)
	}
}
