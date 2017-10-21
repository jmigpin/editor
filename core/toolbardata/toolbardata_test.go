package toolbardata

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestParseTokens1(t *testing.T) {
	s := "a|b|c"
	u := parseTokens(s, 0, len(s), "|")
	if !(len(u) == 3 &&
		u[0].Str == "a" &&
		u[1].Str == "b" &&
		u[2].Str == "c") {
		t.Fatal(spew.Sdump(u))
	}
}
func TestParseTokens2(t *testing.T) {
	s := "\"b|c\""
	u := parseTokens(s, 0, len(s), "|")
	if !(len(u) == 1 &&
		u[0].Str == "b|c") {
		t.Fatal(spew.Sdump(u))
	}
}

func TestParseParts1(t *testing.T) {
	s := "a| \"b | c\" | d"
	u := parseParts(s)
	if !(len(u) == 3 &&
		len(u[1].Args) == 1 &&
		u[1].Args[0].Str == "b | c") {
		t.Fatal(spew.Sdump(u))
	}
}
func TestParseParts2(t *testing.T) {
	s := "a| \\\"b | c\" | d" // first quote is escaped
	u := parseParts(s)
	if !(len(u) == 3 &&
		len(u[1].Args) == 1 &&
		u[1].Args[0].Str == "\\\"b" &&
		len(u[2].Args) == 1 &&
		u[2].Args[0].Str == "c\" | d") {
		t.Fatal(spew.Sdump(u))
	}
}
func TestParseParts3(t *testing.T) {
	s := "|"
	u := parseParts(s)
	if !(len(u) == 1 &&
		len(u[0].Args) == 0) {
		t.Fatal(spew.Sdump(u))
	}
}
func TestParseParts4(t *testing.T) {
	s := "a|b\\|c"
	u := parseParts(s)
	if !(len(u) == 2 &&
		len(u[1].Args) == 1 &&
		u[1].Args[0].Str == "b\\|c") {
		t.Fatal(spew.Sdump(u))
	}
}
func TestParseParts5(t *testing.T) {
	s := `a|b v "" |c`
	u := parseParts(s)
	if !(len(u) == 3 &&
		len(u[1].Args) == 3 &&
		u[1].Args[2].Str == "") {
		t.Fatal(spew.Sdump(u))
	}
}
func TestParseParts6(t *testing.T) {
	s := "a|b v\nc|d"
	u := parseParts(s)
	if !(len(u) == 4 &&
		len(u[2].Args) == 1 &&
		u[2].Args[0].Str == "c") {
		t.Fatal(spew.Sdump(u))
	}
}
