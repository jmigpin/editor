package rwundo

import (
	"testing"

	"github.com/jmigpin/editor/util/iout/iorw"
)

//godebug:annotatepackage

func TestRWUndo1(t *testing.T) {
	s1 := "0123456789"
	rw := iorw.NewBytesReadWriter([]byte(s1))
	h := NewHistory(10)
	rwu := NewRWUndo(rw, h)

	gets := func() string {
		b, _ := iorw.ReadFullFast(rwu)
		return string(b)
	}

	rwu.Overwrite(3, 2, []byte("---")) // "012---56789"
	rwu.Overwrite(7, 0, []byte("+++")) // "012---5+++6789"

	exp := "012---5+++6789"

	s2 := gets()
	if s2 != exp {
		t.Fatal(exp, "got", s2)
	}

	rwu.Undo()
	rwu.Undo()

	s3 := gets()
	if s3 != s1 {
		t.Fatal(s1, "got", s3)
	}

	rwu.Redo()
	rwu.Redo()

	s4 := gets()
	if s4 != s2 {
		t.Fatal(s2, "got", s4)
	}

	rwu.Undo()

	rwu.Overwrite(5, 4, []byte("***"))

	exp2 := "012--***89"
	s5 := gets()
	if s5 != exp2 {
		t.Fatal(exp2, "got", s5)
	}

	rwu.Redo()
	rwu.Redo()
	rwu.Redo()
	s6 := gets()
	if s6 != exp2 {
		t.Fatal(exp2, "got", s6)
	}

	for i := 0; i < 10; i++ {
		rwu.Undo()
	}
	s7 := gets()
	if s7 != s1 {
		t.Fatal(s1, "got", s7)
	}
}

func TestRWUndo2(t *testing.T) {
	s1 := "0123456789"
	rw := iorw.NewBytesReadWriter([]byte(s1))
	h := NewHistory(10)
	rwu := NewRWUndo(rw, h)

	gets := func() string {
		b, _ := iorw.ReadFullFast(rwu)
		return string(b)
	}

	rwu.Overwrite(3, 2, nil) // "01256789"
	rwu.Overwrite(7, 1, nil) // "0125678"
	rwu.Overwrite(4, 1, nil) // "012578"

	exp2 := "012578"
	s2 := gets()
	if s2 != exp2 {
		t.Fatal(exp2, "got", s2)
	}

	rwu.Undo()

	exp3 := "0125678"
	s3 := gets()
	if s3 != exp3 {
		t.Fatal(exp3, "got", s3)
	}
}

func TestRWUndo3(t *testing.T) {
	s1 := "0123456789"
	rw := iorw.NewBytesReadWriter([]byte(s1))
	h := NewHistory(10)
	rwu := NewRWUndo(rw, h)

	gets := func() string {
		b, _ := iorw.ReadFullFast(rwu)
		return string(b)
	}

	rwu.Overwrite(3, 2, nil) // "01256789"
	rwu.History.BeginUndoGroup(nil)
	rwu.Overwrite(7, 1, nil) // "0125678"
	rwu.Overwrite(4, 1, nil) // "012578"
	rwu.History.EndUndoGroup(nil)

	rwu.Undo()

	exp2 := "01256789"
	s2 := gets()
	if s2 != exp2 {
		t.Fatal(exp2, "got", s2)
	}
}

func TestRWUndo4(t *testing.T) {
	s1 := "0123456789"
	rw := iorw.NewBytesReadWriter([]byte(s1))
	h := NewHistory(10)
	rwu := NewRWUndo(rw, h)

	gets := func() string {
		b, _ := iorw.ReadFullFast(rwu)
		return string(b)
	}

	rwu.History.BeginUndoGroup(nil)
	rwu.Overwrite(3, 2, nil) // "01256789"
	rwu.Overwrite(7, 1, nil) // "0125678"
	rwu.Overwrite(4, 1, nil) // "012578"
	rwu.History.EndUndoGroup(nil)

	rwu.Undo()

	exp2 := "0123456789"
	s2 := gets()
	if s2 != exp2 {
		t.Fatal(exp2, "got", s2)
	}
}

func TestRWUndo5(t *testing.T) {
	s1 := "0123456789"
	rw := iorw.NewBytesReadWriter([]byte(s1))
	h := NewHistory(10)
	rwu := NewRWUndo(rw, h)

	gets := func() string {
		b, _ := iorw.ReadFullFast(rwu)
		return string(b)
	}

	rwu.Overwrite(3, 2, nil) // "01256789"
	rwu.Overwrite(7, 1, nil) // "0125678"
	rwu.Overwrite(4, 1, nil) // "012578"

	rwu.Undo()
	rwu.History.ClearUndones()
	rwu.Redo()

	exp2 := "0125678"
	s2 := gets()
	if s2 != exp2 {
		t.Fatal(exp2, "got", s2)
	}
}

func TestRWUndo6(t *testing.T) {
	s1 := "0123456789"
	rw := iorw.NewBytesReadWriter([]byte(s1))
	h := NewHistory(10)
	rwu := NewRWUndo(rw, h)

	gets := func() string {
		b, _ := iorw.ReadFullFast(rwu)
		return string(b)
	}

	rwu.Overwrite(3, 2, nil) // "01256789"
	rwu.Overwrite(7, 1, nil) // "0125678"
	rwu.Overwrite(4, 1, nil) // "012578"

	rwu.Undo()
	rwu.Undo()

	rwu.Overwrite(3, 2, []byte("-")) // "012-789"
	rwu.Overwrite(5, 0, []byte("-")) // "012-7-89"

	exp2 := "012-7-89"
	s2 := gets()
	if s2 != exp2 {
		t.Fatal(exp2, "got", s2)
	}

	//rwu.Redo()
	//rwu.Redo()
	//rwu.Redo()

	//exp3 := exp2
	//s3 := gets()
	//if s3 != exp3 {
	//	t.Fatal(exp3, "got", s3)
	//}

	rwu.Undo()
	rwu.Undo()

	exp4 := "01256789"
	s4 := gets()
	if s4 != exp4 {
		t.Fatal(exp4, "got", s4)
	}

	rwu.Undo()

	s5 := gets()
	if s5 != s1 {
		t.Fatal(s1, "got", s5)
	}
}
