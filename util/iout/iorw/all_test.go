package iorw

import (
	"bytes"
	"context"
	"testing"
	"unicode"
)

func TestRW1(t *testing.T) {
	s := "0123"
	rw := NewBytesReadWriter([]byte(s))

	type ins struct {
		i int
		s string
		e string
	}
	type del struct {
		i int
		l int
		e string
	}
	type ow struct {
		i int
		l int
		s string
		e string
	}

	var tests = []interface{}{
		&ins{1, "ab", "0ab123"},
		&ins{5, "ab", "0ab12ab3"},
		&del{1, 2, "012ab3"},
		&del{3, 2, "0123"},
		&ins{1, "ab", "0ab123"},
		//&ow{0, 6, "abcde", "abcde"},
		//&ow{0, 5, "abc", "abc"},
		//&ow{0, 1, "abc", "abcbc"},
	}

	for _, u := range tests {
		switch w := u.(type) {
		case *ins:
			if err := rw.Insert(w.i, []byte(w.s)); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(rw.buf, []byte(w.e)) {
				t.Fatal(string(rw.buf))
			}
		case *del:
			if err := rw.Delete(w.i, w.l); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(rw.buf, []byte(w.e)) {
				t.Fatal(string(rw.buf))
			}
		//case *ow:
		//	if err := rw.Overwrite(w.i, w.l, []byte(w.s)); err != nil {
		//		t.Fatal(err)
		//	}
		//	if !bytes.Equal(rw.buf, []byte(w.e)) {
		//		t.Fatal(string(rw.buf) + " != " + w.e)
		//	}
		default:
			t.Fatal("bad type")
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

	rw := NewBytesReadWriter([]byte(s))

	i, err := Index(rw, 4, []byte("abc"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(i)
}

func TestIndex2(t *testing.T) {
	s := "012345678"
	rw := NewBytesReadWriter([]byte(s))
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
	rw := NewBytesReadWriter([]byte(s))

	fn := func(ru rune) bool {
		return ru == '\n'
	}

	i, _, err := LastIndexFunc(rw, 6, true, fn)
	if err != nil {
		t.Fatal(err)
	}
	if i != 1 {
		t.Fatal(i)
	}
}

func TestExpandIndex1(t *testing.T) {
	s := "a 234 b"
	rw := NewBytesReadWriter([]byte(s))
	i := ExpandIndexFunc(rw, 3, true, unicode.IsSpace)
	if i != 5 {
		t.Fatal(i)
	}
	i = ExpandIndexFunc(rw, i+1, true, unicode.IsSpace)
	if i != 7 {
		t.Fatal(i)
	}
}

func TestExpandLastIndex1(t *testing.T) {
	s := "a 234 b"
	rw := NewBytesReadWriter([]byte(s))
	i := ExpandLastIndexFunc(rw, 3, true, unicode.IsSpace)
	if i != 2 {
		t.Fatal(i)
	}
	// repeat from same position
	i = ExpandLastIndexFunc(rw, i, true, unicode.IsSpace)
	if i != 2 {
		t.Fatal(i)
	}

	i = ExpandLastIndexFunc(rw, i-1, true, unicode.IsSpace)
	if i != 0 {
		t.Fatal(i)
	}
}
