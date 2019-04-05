package toolbarparser

import (
	"testing"
)

func TestParseVar1(t *testing.T) {
	s1 := "~0=\"a b c\""
	v, err := ParseVar(s1)
	if err != nil {
		t.Fatal(err)
	}
	if v.Value != "a b c" {
		t.Fatal(v)
	}
}

//----------

func TestParseVar2(t *testing.T) {
	s1 := "$abc=0123"
	v, err := ParseVar(s1)
	if err != nil {
		t.Fatal(err)
	}
	if !(v.Name == "$abc" && v.Value == "0123") {
		t.Fatal(v)
	}
}

func TestParseVar3(t *testing.T) {
	s1 := "$abc"
	v, err := ParseVar(s1)
	if err != nil {
		t.Fatal(err)
	}
	if !(v.Name == "$abc" && v.Value == "") {
		t.Fatal(v)
	}
}

//----------

func testMap() VarMap {
	return VarMap{
		"~":  "/a/b/c",
		"~0": "~/d/e/",
		"~1": "~0/f",
	}
}

func TestEncode1(t *testing.T) {
	em := testMap()
	s1 := "/a/b/c/d/e/f/g.txt"
	s2 := "~1/g.txt"
	r1 := EncodeVars(s1, em)
	if r1 != s2 {
		t.Fatal(r1)
	}
	r2 := DecodeVars(r1, em)
	if r2 != s1 {
		t.Fatal(r2)
	}
}

func TestDecode1(t *testing.T) {
	em := testMap()
	s1 := "/a/b/c/d/e/f/g.txt"
	s2 := "~0/f/g.txt"
	r2 := DecodeVars(s2, em)
	if r2 != s1 {
		t.Fatal(r2)
	}
}
