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
func TestParseDeclVar5(t *testing.T) {
	s1 := "$ábc=1"
	_, err := parseVarDecl(s1)
	if err == nil {
		t.Fatal("expecting error")
	}
}
func TestParseDeclVar6(t *testing.T) {
	s1 := "$ab-c=1"
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

func TestDecodeLeadingHomeVarParentDoesNotMatchChildVar(t *testing.T) {
	vm := VarMap{
		"~":  "/home/a",
		"~0": "~/workspace/project",
		"~1": "~0/cmd/app/web",
	}
	hvm := NewHomeVarMap(vm, false)

	parent := hvm.Decode("~0/../")
	child := hvm.Decode("~1")

	if parent != "/home/a/workspace" {
		t.Fatal(parent)
	}
	if child != "/home/a/workspace/project/cmd/app/web" {
		t.Fatal(child)
	}
	if parent == child {
		t.Fatalf("parent and child should not match: %q", parent)
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

func TestParseVarsDecodeLeadingHomeVars(t *testing.T) {
	hvm := NewHomeVarMap(VarMap{
		"~":  "/home/a",
		"~1": "/tmp/project",
	}, false)

	data := Parse("$p=~1/src | $home=~/bin | $url=http://x/~1 | run")
	vm := hvm.ParseAndDecodeVars(data)

	if vm["$p"] != "/tmp/project/src" {
		t.Fatal(vm)
	}
	if vm["$home"] != "/home/a/bin" {
		t.Fatal(vm)
	}
	if vm["$url"] != "http://x/~1" {
		t.Fatal(vm)
	}
}

func TestParseVarRefs1(t *testing.T) {
	s := []byte("$aaa x ~{1} y ${bb}")
	vrs, err := parseVarRefs(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(vrs) != 3 {
		t.Fatal(len(vrs), vrs)
	}
	if vrs[0].Name != "$aaa" || vrs[0].SrcString(s) != "$aaa" {
		t.Fatal(vrs[0])
	}
	if vrs[1].Name != "~1" || vrs[1].SrcString(s) != "~{1}" {
		t.Fatal(vrs[1])
	}
	if vrs[2].Name != "$bb" || vrs[2].SrcString(s) != "${bb}" {
		t.Fatal(vrs[2])
	}
}

func TestParseVarRefs2(t *testing.T) {
	s := []byte("\\$aaa \"$bbb\" $ccc")
	vrs, err := parseVarRefs(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(vrs) != 1 {
		t.Fatal(len(vrs), vrs)
	}
	if vrs[0].Name != "$ccc" {
		t.Fatal(vrs[0])
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
