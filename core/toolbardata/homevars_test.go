package toolbardata

import "testing"

func TestHomeVars1(t *testing.T) {
	hv := HomeVars{}
	hv.Append("1", "a/b/c")
	s := hv.Encode("a/b/ceee")
	if s != "1eee" {
		t.Fatal(s)
	}
}
func TestHomeVars2(t *testing.T) {
	hv := HomeVars{}
	hv.Append("1", "abc")
	hv.Append("2", "1de")

	a, b := "2", "abcde"
	s := hv.Decode(a)
	if s != b {
		t.Fatal(s)
	}
	s = hv.Encode(b)
	if s != a {
		t.Fatal(s)
	}
}
func TestHomeVars3(t *testing.T) {
	hv := HomeVars{}
	hv.Append("1", "abc")
	hv.Append("2", "1de")
	hv.Append("3", "1")
	hv.Append("1", "3")

	a, b := "1abc", "abcabc"
	s := hv.Decode(a)
	if s != b {
		t.Fatal(s)
	}
	s = hv.Encode(b)
	if s != a {
		t.Fatal(s)
	}
}
func TestHomeVars4(t *testing.T) {
	hv := HomeVars{}
	hv.Append("1", "abc")
	hv.Append("2", "")
	hv.Append("", "3")

	a, b := "1abc", "abcabc"
	s := hv.Decode(a)
	if s != b {
		t.Fatal(s)
	}
	s = hv.Encode(b)
	if s != a {
		t.Fatal(s)
	}
}
