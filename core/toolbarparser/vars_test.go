//go:build !windows

package toolbarparser

import (
	"testing"
)

func TestParseVarDecl1(t *testing.T) {
	s1 := "~0=\"a b c\""
	v, err := parseVarDecl(s1)
	if err != nil {
		t.Fatal(err)
	}
	if v.Value != "\"a b c\"" {
		t.Fatal(v)
	}
}
func TestParseDeclVar1b(t *testing.T) {
	s1 := "~1" // value can't be empty
	_, err := parseVarDecl(s1)
	if err == nil {
		t.Fatal(err)
	}
}
func TestParseDeclVar1c(t *testing.T) {
	s1 := "~1=c"
	_, err := parseVarDecl(s1)
	if err != nil {
		t.Fatal(err)
	}
}
func TestParseVarDecl2(t *testing.T) {
	s1 := "$abc=0123"
	v, err := parseVarDecl(s1)
	if err != nil {
		t.Fatal(err)
	}
	if !(v.Name == "$abc" && v.Value == "0123") {
		t.Fatal(v)
	}
}
func TestParseDeclVar3(t *testing.T) {
	s1 := "$abc="
	v, err := parseVarDecl(s1)
	if err != nil {
		t.Fatal(err)
	}
	if !(v.Name == "$abc" && v.Value == "") {
		t.Fatal(v)
	}
}
func TestParseDeclVar4(t *testing.T) {
	s1 := "$abc"
	_, err := parseVarDecl(s1)
	if err == nil {
		t.Fatal("expecting error")
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

//----------

func TestParseVars1(t *testing.T) {
	s := "$aaa=1 | $bb=2 | $c=3${aaa}4$bb+ | ~1=5 | $d=~0~{1}2"
	d := Parse(s)
	vm := ParseVars(d)
	if v, ok := vm["$c"]; !ok || v != "3142+" {
		t.Fatal(vm)
	}
	if v, ok := vm["$d"]; !ok || v != "~052" {
		t.Fatal(vm)
	}
}
func TestParseVars2(t *testing.T) {
	s := "$a=1 | $b=\"2$a\"$a"
	d := Parse(s)
	vm := ParseVars(d)
	if v, ok := vm["$b"]; !ok || v != "\"2$a\"1" {
		t.Fatal(vm)
	}
}
func TestParseVars3(t *testing.T) {
	s := "~1=abc | $a=~~1"
	d := Parse(s)
	vm := ParseVars(d)
	if v, ok := vm["$a"]; !ok || v != "~abc" {
		t.Fatal(vm)
	}
}

//----------

var benchStr1 = "$aaa=b | $a=a${aaa}c+$aaa+| ~1=zzz | $c=~1"

func BenchmarkParseVars1(b *testing.B) {
	s := benchStr1 + benchStr1
	d := Parse(s)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm := ParseVars(d)
		_ = vm
	}
}
