package parseutil

import (
	"testing"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func TestParseResource1(t *testing.T) {
	s := "AAA /a/b/c.txt AAA"
	testParseResourcePath(t, s, 10, "/a/b/c.txt")
}

func TestParseResource2(t *testing.T) {
	s := "AAA /a/b/c%20c.txt AAA"
	testParseResourcePath(t, s, 9, "/a/b/c c.txt")
}

func TestParseResource3(t *testing.T) {
	s := "AAA /a/b/c\\ c.txt AAA"
	testParseResourcePath(t, s, 9, "/a/b/c c.txt")
}

func TestParseResource4(t *testing.T) {
	s := "AAA /a/b/c\\\\ c.txt AAA"
	testParseResourcePath(t, s, 9, "/a/b/c\\")
}

func TestParseResource5(t *testing.T) {
	s := "AAA /a/b/c\\ c.txt AAA"
	testParseResourcePath(t, s, 11, "/a/b/c c.txt") // index in mid escape
}

func TestParseResource5_2(t *testing.T) {
	s := "AAA /a/b/c\\ c.txt AAA"
	testParseResourcePath(t, s, 12, "/a/b/c c.txt")
}

func TestParseResource5_3(t *testing.T) {
	s := "AAA /a/b/c\\ c.txt AAA"
	testParseResourcePath(t, s, 13, "/a/b/c c.txt")
}

func TestParseResource5_4(t *testing.T) {
	s := "/a/b/c\\ c.txt AAA"
	testParseResourcePath(t, s, 0, "/a/b/c c.txt")
}

func TestParseResource5_5(t *testing.T) {
	s := " a\\ b\\ c.txt"
	testParseResourcePath(t, s, 3, "a b c.txt")
}

func TestParseResource6(t *testing.T) {
	s := "AAA /a/b/c.txt\\:a:1:2# AAA"
	testParseResourcePath(t, s, 11, "/a/b/c.txt:a")
	testParseResourceLineCol(t, s, 11, 1, 2)
}

func TestParseResource7(t *testing.T) {
	s := "AAA /a/b/c.txt\\:a:1:#AAA"
	testParseResourcePath(t, s, 11, "/a/b/c.txt:a")
	testParseResourceLineCol(t, s, 11, 1, 0)
}

func TestParseResource8(t *testing.T) {
	s := "/a/b/c:1:2"
	testParseResourcePath(t, s, 0, "/a/b/c")
	testParseResourceLineCol(t, s, 0, 1, 2)
}

func TestParseResource9(t *testing.T) {
	s := "/a/b\\ b/c"
	testParseResourcePath(t, s, 0, "/a/b b/c")
}

func TestParseResource10(t *testing.T) {
	s := "/a/b\\"
	testParseResourcePath(t, s, 0, "/a/b")
}

func TestParseResource11(t *testing.T) {
	s := ": /a/b/c"
	testParseResourcePath(t, s, len(s), "/a/b/c")
}

func TestParseResource12(t *testing.T) {
	s := "//a/b/////c"
	testParseResourcePath(t, s, len(s), "/a/b/c")
}

func TestParseResource13(t *testing.T) {
	s := "(/a/b/c.txt)"
	testParseResourcePath(t, s, 5, "/a/b/c.txt")
}

func TestParseResource14(t *testing.T) {
	s := "[/a/b/c.txt]"
	testParseResourcePath(t, s, 5, "/a/b/c.txt")
}

func TestParseResource15(t *testing.T) {
	s := "</a/b/c.txt>"
	testParseResourcePath(t, s, 5, "/a/b/c.txt")
}

func TestParseResource16(t *testing.T) {
	s := ""
	rw := iorw.NewBytesReadWriter([]byte(s))
	_, err := ParseResource(rw, 0)
	if err == nil {
		t.Fatal("able to parse empty string")
	}
}

func TestParseResource17(t *testing.T) {
	s := "./a/b/c.txt :20"
	testParseResourcePath(t, s, 3, "./a/b/c.txt")
	testParseResourceLineCol(t, s, 0, 0, 0)
}

func TestParseResource18(t *testing.T) {
	s := "aa file:///a/b/c.txt bb"
	testParseResourcePath(t, s, 3, "/a/b/c.txt")
	testParseResourcePath(t, s, 10, "/a/b/c.txt")
	testParseResourcePath(t, s, 15, "/a/b/c.txt")
}

func TestParseResource19(t *testing.T) {
	s := "aa &{file:///a/b/c.txt}"
	testParseResourcePath(t, s, 15, "/a/b/c.txt")
}

func TestParseResource20(t *testing.T) {
	s := "aa &{file:///a/b/c%2b%2b.txt}"
	testParseResourcePath(t, s, 15, "/a/b/c++.txt")
}

func TestParseResource21(t *testing.T) {
	s := "-arg=/a/b/c.txt"
	testParseResourcePath(t, s, 8, "/a/b/c.txt")
}

//----------

func testParseResourcePath(t *testing.T, str string, index int, estr string) {
	t.Helper()
	rw := iorw.NewBytesReadWriter([]byte(str))
	u, err := ParseResource(rw, index)
	if err != nil {
		t.Fatal(err)
	}
	if u.Path != estr {
		t.Fatalf("%#v", u)
	}
}

func testParseResourceLineCol(t *testing.T, str string, index int, eline, ecol int) {
	t.Helper()
	rw := iorw.NewBytesReadWriter([]byte(str))
	u, err := ParseResource(rw, index)
	if err != nil {
		t.Fatal(err)
	}
	if u.Line != eline || u.Column != ecol {
		t.Fatalf("%v\n%#v", str, u)
	}
}

//----------

func TestExpand1(t *testing.T) {
	s := ": /a/b/c"
	rw := iorw.NewBytesReadWriter([]byte(s))
	l, _ := ExpandIndexesEscape(rw, rw.Max(), false, isResourceRune, '\\')
	if l != 2 {
		t.Fatalf("%v", l)
	}
}

//----------

func TestIndexLineColumn1(t *testing.T) {
	s := "123\n123\n123"
	rw := iorw.NewBytesReadWriter([]byte(s))
	l, c, err := IndexLineColumn(rw, 0)
	if err != nil {
		t.Fatal(err)
	}
	i, err := LineColumnIndex(rw, l, c)
	if err != nil {
		t.Fatal(err)
	}
	if i != 0 {
		t.Fatal(i, rw.Max())
	}
}
func TestIndexLineColumn2(t *testing.T) {
	s := "123\n123\n123"
	rw := iorw.NewBytesReadWriter([]byte(s))
	l, c, err := IndexLineColumn(rw, rw.Max())
	if err != nil {
		t.Fatal(err)
	}
	i, err := LineColumnIndex(rw, l, c)
	if err != nil {
		t.Fatal(err)
	}
	if i != rw.Max() {
		t.Fatal(i, rw.Max())
	}
}

func TestLineColumnIndex1(t *testing.T) {
	s := "123\n123\n123"
	rw := iorw.NewBytesReadWriter([]byte(s))
	i, err := LineColumnIndex(rw, 3, 10)
	if err != nil {
		t.Fatal(err)
	}
	if i != 8 { // beginning of line
		t.Fatal(i, rw.Max())
	}
}

//----------

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

//----------

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
