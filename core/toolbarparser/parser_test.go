//go:build !windows
// +build !windows

package toolbarparser

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
)

//godebug:annotatepackage
//godebug:annotatepackage:github.com/jmigpin/editor/util/parseutil/lrparser

func TestParseTokens1(t *testing.T) {
	s := "a|b|c"
	d := Parse(s)
	if !(len(d.Parts) == 3 &&
		d.Parts[0].Str() == "a" &&
		d.Parts[1].Str() == "b" &&
		d.Parts[2].Str() == "c") {
		t.Fatal(spew.Sdump(d))
	}
}
func TestParseTokens2(t *testing.T) {
	s := "\"b|c\""
	d := Parse(s)
	if !(len(d.Parts) == 1 &&
		d.Parts[0].Str() == "\"b|c\"") {
		t.Fatal(spew.Sdump(d))
	}
}
func TestParseParts3(t *testing.T) {
	s := "|"
	d := Parse(s)
	if !(len(d.Parts) == 2 &&
		len(d.Parts[0].Args) == 0 &&
		len(d.Parts[1].Args) == 0) {
		t.Fatal(spew.Sdump(d))
	}
}
func TestParseParts4(t *testing.T) {
	s := "a|b\\|c"
	d := Parse(s)
	if !(len(d.Parts) == 2 &&
		len(d.Parts[1].Args) == 1 &&
		d.Parts[1].Args[0].Str() == "b\\|c") {
		t.Fatal(spew.Sdump(d))
	}
}
func TestParseParts5(t *testing.T) {
	s := ` a|b v "" | c`
	d := Parse(s)
	if !(len(d.Parts) == 3 &&
		len(d.Parts[1].Args) == 3 &&
		d.Parts[1].Args[2].Str() == "\"\"") {
		//t.Fatal(spew.Sdump(d))
		t.Fatal(d)
	}
}
func TestParseParts6(t *testing.T) {
	s := "a|b v\nc|d"
	d := Parse(s)
	if !(len(d.Parts) == 4 &&
		len(d.Parts[2].Args) == 1 &&
		d.Parts[2].Args[0].Str() == "c") {
		t.Fatal(spew.Sdump(d))
	}
}
func TestParseParts7(t *testing.T) {
	s := `  grep -niIR      "== last"  `
	d := Parse(s)
	if !(len(d.Parts) == 1 &&
		len(d.Parts[0].Args) == 3 &&
		d.Parts[0].Args[0].Str() == `grep` &&
		d.Parts[0].Args[1].Str() == `-niIR` &&
		d.Parts[0].Args[2].Str() == `"== last"`) {
		//t.Fatal(spew.Sdump(d))
		t.Fatal(d)
	}
}

func TestParseParts8(t *testing.T) {
	s := "\"a\"|'b'|`c\nd`|\"e\\\"\""
	d := Parse(s)

	{
		s1 := d.Parts[0].Args[0].Str()
		s2 := d.Parts[0].Args[0].UnquotedStr()
		if !(s1 == "\"a\"" && s2 == "a") {
			t.Fatal(spew.Sdump(d.Parts[0].Args[0].UnquotedStr(), d.Parts[0]))
		}
	}
	{
		s1 := d.Parts[1].Args[0].Str()
		s2 := d.Parts[1].Args[0].UnquotedStr()
		if !(s1 == "'b'" && s2 == "b") {
			t.Fatal(spew.Sdump(d.Parts[1]))
		}
	}
	{
		s1 := d.Parts[2].Args[0].Str()
		s2 := d.Parts[2].Args[0].UnquotedStr()
		if !(s1 == "`c\nd`" && s2 == "c\nd") {
			//t.Fatal(spew.Sdump(d.Parts[2]))
			t.Fatal(s1, s2)
			//d.bnd.PrintRuleTree(5)
			//t.Fatal()
		}
	}
	{
		s1 := d.Parts[3].Args[0].Str()
		s2 := d.Parts[3].Args[0].UnquotedStr()
		if !(s1 == "\"e\\\"\"" && s2 == "e\"") {
			t.Fatal(spew.Sdump(d.Parts[2]))
		}
	}
}

func TestParseParts9(t *testing.T) {
	s := `|||`
	d := Parse(s)
	if !(len(d.Parts) == 4 &&
		len(d.Parts[0].Args) == 0 &&
		len(d.Parts[1].Args) == 0) {
		t.Fatal(spew.Sdump(d))
	}
}

func TestParseParts10(t *testing.T) {
	s := `|\`
	d := Parse(s)
	if !(len(d.Parts) == 2 &&
		len(d.Parts[0].Args) == 0 &&
		len(d.Parts[1].Args) == 1) {
		t.Fatal(spew.Sdump(d))
	}
}

func TestParseParts11(t *testing.T) {
	s := `|"\""|`
	d := Parse(s)
	if !(len(d.Parts) == 3 &&
		len(d.Parts[0].Args) == 0 &&
		len(d.Parts[1].Args) == 1) {
		t.Fatal(spew.Sdump(d))
	}
}

func TestParseParts12(t *testing.T) {
	s := `aa"bbb|"ccc`
	d := Parse(s)
	if !(len(d.Parts) == 1 &&
		len(d.Parts[0].Args) == 1) {
		t.Fatal(spew.Sdump(d))
	}
}

func TestParseParts13(t *testing.T) {
	s := `aa"bbb|ccc`
	d := Parse(s)
	if !(len(d.Parts) == 2 &&
		len(d.Parts[0].Args) == 1 &&
		len(d.Parts[1].Args) == 1) {
		t.Fatal(spew.Sdump(d))
	}
}

func TestParseParts14(t *testing.T) {
	s := `a\ b|c`
	d := Parse(s)
	if !(len(d.Parts) == 2 &&
		len(d.Parts[0].Args) == 1 &&
		len(d.Parts[1].Args) == 1) {
		t.Fatal(spew.Sdump(d))
	}
}

func TestParseParts15(t *testing.T) {
	s := "aa\"bbb\n|\"ccc" // double quote doesn't accept newline
	d := Parse(s)
	if !(len(d.Parts) == 3 &&
		len(d.Parts[0].Args) == 1 &&
		len(d.Parts[1].Args) == 0 &&
		len(d.Parts[2].Args) == 1) {
		t.Fatal(spew.Sdump(d))
	}
}

func TestParseParts16(t *testing.T) {
	s := "a\\ aa\\|aa|bb"
	d := Parse(s)
	if len(d.Parts) != 2 {
		t.Fatal()
	}
	str1 := d.Parts[0].Str()
	if str1 != "a\\ aa\\|aa" {
		t.Fatal()
	}
	str2 := d.Parts[1].Str()
	if str2 != "bb" {
		t.Fatal()
	}
}
