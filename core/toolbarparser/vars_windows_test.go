package toolbarparser

import (
	"testing"
)

func TestEncDec3(t *testing.T) {
	vm := VarMap{
		"~":  "c:\\a\\b",
		"~1": "c:\\A\\b",
		"~2": "c:\\a\\b",
	}
	hvm := NewHomeVarMap(vm, true)
	s1 := "C:\\a\\b\\c.txt"
	s2 := "~\\c.txt"
	for i := 0; i < 100; i++ {
		r1 := hvm.Encode(s1)
		if r1 != s2 {
			t.Fatalf("i=%v, s1=%v, r1=%v", i, r1, s2)
		}
	}
}

func TestEncDec4(t *testing.T) {
	vm := VarMap{
		"~": "c:\\a\\b",
	}
	hvm := NewHomeVarMap(vm, true)
	s1 := "C:\\A\\b\\C.txt"
	s2 := "~\\C.txt"
	for i := 0; i < 100; i++ {
		r1 := hvm.Encode(s1)
		if r1 != s2 {
			t.Fatalf("i=%v, s1=%v, r1=%v", i, r1, s2)
		}
	}
}
