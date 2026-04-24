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
	s := ""
	for i := 0; i < 10; i++ {
		s += "0123456789"
	}
	s += "abc"

	rw := NewStringReaderAt(s)

	i, _, err := indexCtx2(context.Background(), rw, 0, []byte("abc"), 4, &IndexOpt{IgnoreCase: true})
	if err != nil {
		t.Fatal(err)
	}
	if i < 0 {
		t.Fatal("not found")
	}
	t.Log(i)
}

func TestIndex2(t *testing.T) {
	s := "0123456789"
	rw := NewStringReaderAt(s)
	i, _, err := indexCtx2(context.Background(), rw, 0, []byte("345"), 4, &IndexOpt{IgnoreCase: true})
	if err != nil {
		t.Fatal(err)
	}
	if i < 0 {
		t.Fatal("not found")
	}
}

func TestLastIndex1(t *testing.T) {
	s := "-abc"
	for i := 0; i < 10; i++ {
		s += "0123456789"
	}
	s += ""

	rw := NewStringReaderAt(s)

	i, _, err := lastIndexCtx2(context.Background(), rw, rw.Max(), []byte("abc"), 4, &IndexOpt{IgnoreCase: true})
	if err != nil {
		t.Fatal(err)
	}
	if i < 0 {
		t.Fatal("not found")
	}
	t.Log(i)
}

func TestLastIndex2(t *testing.T) {
	s := "-abc----------abc----------"

	rw := NewStringReaderAt(s)

	i, _, err := lastIndexCtx2(context.Background(), rw, rw.Max(), []byte("abc"), 4, &IndexOpt{IgnoreCase: true})
	if err != nil {
		t.Fatal(err)
	}
	if i != 14 {
		t.Fatal(i)
	}

	i, _, err = lastIndexCtx2(context.Background(), rw, i, []byte("abc"), 4, &IndexOpt{IgnoreCase: true})
	if err != nil {
		t.Fatal(err)
	}
	if i != 1 {
		t.Fatal(i)
	}
}

func TestIndexDiacritics1(t *testing.T) {
	s := "-ìùù-aaáéb--"
	rw := NewStringReaderAt(s)
	i, _, err := indexCtx2(context.Background(), rw, 0, []byte("aae"), -1, &IndexOpt{IgnoreCase: true, IgnoreDiacritics: true})
	if err != nil {
		t.Fatal(err)
	}
	if i < 0 {
		t.Fatal("not found")
	}
	t.Log(i)
}

func TestIndexDiacritics2(t *testing.T) {
	s := "úúú-aáé--"
	rw := NewStringReaderAt(s)
	i, n, err := indexCtx2(context.Background(), rw, 0, []byte("aae"), -1, &IndexOpt{IgnoreCase: true, IgnoreDiacritics: true})
	if err != nil {
		t.Fatal(err)
	}
	if i < 0 {
		t.Fatal("not found")
	}
	if i != 7 {
		t.Fatal(i)
	}
	if n != 5 {
		t.Fatal(n)
	}
}

func TestIndexDiacritics3(t *testing.T) {
	s := "úúú-aáé--ú"
	rw := NewStringReaderAt(s)
	i, n, err := lastIndexCtx2(context.Background(), rw, rw.Max(), []byte("aae"), -1, &IndexOpt{IgnoreCase: true, IgnoreDiacritics: true})
	if err != nil {
		t.Fatal(err)
	}
	if i < 0 {
		t.Fatal("not found")
	}
	if i != 7 {
		t.Fatal(i)
	}
	if n != 5 {
		t.Fatal(n)
	}
}

func TestIndexDiacritics4(t *testing.T) {
	s := "úúú-aáé--ú"
	rw := NewStringReaderAt(s)
	i, n, err := indexCtx2(context.Background(), rw, 0, []byte("aae"), -1, &IndexOpt{IgnoreCase: true, IgnoreDiacritics: true})
	if err != nil {
		t.Fatal(err)
	}
	if i < 0 {
		t.Fatal("not found")
	}
	if i != 7 {
		t.Fatal(i)
	}
	if n != 5 {
		t.Fatal(n)
	}
}

func TestIndexDiacritics5(t *testing.T) {
	s := "úúú-aÁé--ú"
	rw := NewStringReaderAt(s)
	i, n, err := indexCtx2(context.Background(), rw, 0, []byte("aáé"), -1, &IndexOpt{IgnoreCase: true, IgnoreCaseDiacritics: true})
	if err != nil {
		t.Fatal(err)
	}
	if i < 0 {
		t.Fatal("not found")
	}
	if i != 7 {
		t.Fatal(i)
	}
	if n != 5 {
		t.Fatal(n)
	}
}

func TestPrepareForCompare(t *testing.T) {
	fn1 := prepareForCompareFn(&IndexOpt{IgnoreCase: true, IgnoreDiacritics: true})
	s1 := "ááBC"
	p, _, err := fn1([]byte(s1))
	if err != nil {
		t.Fatal(err)
	}
	s2 := string(p)
	if s2 != "aabc" {
		t.Fatal(s2)
	}
}

//----------

func BenchmarkPrepareForCompare(b *testing.B) {
	pfcFn := prepareForCompareFn(&IndexOpt{IgnoreCase: true, IgnoreDiacritics: true})
	benchmarkPrepareForCompare2(b, pfcFn)
}
func benchmarkPrepareForCompare2(b *testing.B, pfcFn pfcType) {
	s3 := "Áńkúabc"
	for i := 0; i < 15; i++ {
		s3 += s3
	}
	b.ResetTimer()
	for k := 0; k < b.N; k++ {
		p, _, err := pfcFn([]byte(s3))
		if err != nil {
			b.Fatal(err)
		}
		_ = p
	}
}

//----------

func TestRuneLastIndex1(t *testing.T) {
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
