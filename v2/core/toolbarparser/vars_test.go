// +build !windows

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

func TestParseVar1_2(t *testing.T) {
	s1 := "~1" // value can't be empty
	_, err := ParseVar(s1)
	if err == nil {
		t.Fatal(err)
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

func TestEncode1(t *testing.T) {
	vm := VarMap{
		"~":  "/a/b/c",
		"~0": "~/d/e/",
		"~1": "~0/f",
	}
	hvm := NewHomeVarMap(vm, false)
	s1 := "/a/b/c/d/e/f/g.txt"
	s2 := "~1/g.txt"
	r1 := hvm.Encode(s1)
	if r1 != s2 {
		t.Fatal(r1)
	}
	r2 := hvm.Decode(r1)
	if r2 != s1 {
		t.Fatal(r2)
	}
}

func TestDecode1(t *testing.T) {
	vm := VarMap{
		"~":  "/a/b/c",
		"~0": "~/d/e/",
		"~1": "~0/f",
	}
	hvm := NewHomeVarMap(vm, false)
	s1 := "/a/b/c/d/e/f/g.txt"
	s2 := "~0/f/g.txt"
	r2 := hvm.Decode(s2)
	if r2 != s1 {
		t.Fatal(r2)
	}
}

//----------

func TestEncDec1(t *testing.T) {
	vm := VarMap{
		"~":  "/a/b", // same value as ~1
		"~1": "/a/b",
		"~0": "~/c/d",
	}
	hvm := NewHomeVarMap(vm, false)
	s1 := "/a/b"
	for i := 0; i < 100; i++ {
		r1 := hvm.Encode(s1)
		if r1 != "~" {
			t.Fatalf("i=%v, s1=%v, r1=%v", i, s1, r1)
		}
	}
}

func TestEncDec2(t *testing.T) {
	vm := VarMap{
		"~":  "/a/b",
		"~1": "~/c",
		"~2": "/a/b/c", // same as "~" +"~1"
	}
	hvm := NewHomeVarMap(vm, false)
	s1 := "/a/b/c/d.txt"
	s2 := "~1/d.txt"
	for i := 0; i < 100; i++ {
		r1 := hvm.Encode(s1)
		if r1 != s2 {
			t.Fatalf("i=%v, s1=%v, r1=%v", i, r1, s2)
		}
	}
}
