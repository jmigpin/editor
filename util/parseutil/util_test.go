package parseutil

import (
	"testing"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func TestDetectVar(t *testing.T) {
	str := "aaaa$b $cd $e"
	if !DetectEnvVar(str, "b") {
		t.Fatal()
	}
	if !DetectEnvVar(str, "cd") {
		t.Fatal()
	}
	if !DetectEnvVar(str, "e") {
		t.Fatal()
	}

	str2 := "$a"
	if !DetectEnvVar(str2, "a") {
		t.Fatal()
	}
}

func TestAddEscapes(t *testing.T) {
	s := "a \\b"
	s2 := AddEscapes(s, '\\', " \\")
	if s2 != "a\\ \\\\b" {
		t.Fatal()
	}
	s3 := RemoveEscapes(s2, '\\')
	if s3 != s {
		t.Fatal()
	}
}

func TestIndexLineColumn1(t *testing.T) {
	s := "123\n123\n123"
	rd := iorw.NewStringReaderAt(s)
	l, c, err := IndexLineColumn(rd, 0)
	if err != nil {
		t.Fatal(err)
	}
	i, err := LineColumnIndex(rd, l, c)
	if err != nil {
		t.Fatal(err)
	}
	if i != 0 {
		t.Fatal(i, rd.Max())
	}
}
func TestIndexLineColumn2(t *testing.T) {
	s := "123\n123\n123"
	rd := iorw.NewStringReaderAt(s)
	l, c, err := IndexLineColumn(rd, rd.Max())
	if err != nil {
		t.Fatal(err)
	}
	i, err := LineColumnIndex(rd, l, c)
	if err != nil {
		t.Fatal(err)
	}
	if i != rd.Max() {
		t.Fatal(i, rd.Max())
	}
}

func TestLineColumnIndex1(t *testing.T) {
	s := "123\n123\n123"
	rw := iorw.NewStringReaderAt(s)
	i, err := LineColumnIndex(rw, 3, 10)
	if err != nil {
		t.Fatal(err)
	}
	if i != 8 { // beginning of line
		t.Fatal(i, rw.Max())
	}
}
