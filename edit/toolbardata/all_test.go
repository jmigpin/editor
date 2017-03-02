package toolbardata

import (
	"fmt"
	"testing"
)

//func TestToolbarTextData1(t *testing.T) {
//s := "cmd1|cmd2|cmd3"
//td := NewToolbarTextData(s)
//p := td.Parts
//if !(len(p) == 3 && p[0] == "cmd1" && p[1] == "cmd2" && p[2] == "cmd3") {
//t.Fatalf("items: %s\n", td.Parts)
//}
//}
//func TestToolbarTextData2(t *testing.T) {
//s := "cmd1|cmd2\\|cmd3"
//td := NewToolbarTextData3(s)
//p := td.Parts
//if !(len(p) == 2 && p[0] == "cmd1" && p[1] == "cmd2|cmd3") {
//t.Fatalf("parts: %s\n", td.Parts)
//}
//}
//func TestToolbarTextDataTags1(t *testing.T) {
//s := "a|b|c|c:a\\|b"
//td := NewToolbarTextData3(s)
//if s, ok := td.GetTag("c"); !(ok && s == "a|b") {
//t.Fatalf("missing command tag: %v", td)
//}
//}
//func TestToolbarTextDataTags2(t *testing.T) {
//s := "a|b|c|f:name1"
//td := NewToolbarTextData3(s)
//if s, ok := td.FilenameTag(); !(ok && s == "name1") {
//t.Fatalf("missing filename tag: %v", td)
//}
//}
//func TestToolbarTextDataTagsPosition(t *testing.T) {
//s := "a|b|c|f:name1"
//td := NewToolbarTextData3(s)
//if s, _, _, ok := td.GetPartAtIndex(4); !(ok && s == "c") {
//t.Fatalf("failed to get part: %v", td)
//}
//}
//func TestToolbarTextDataStrings(t *testing.T) {
//s := "a|c:cmd -c \"e|f\" | d"
//td := NewToolbarTextData3(s)
//if !(len(td.Parts[1]) >= 2 && td.Parts[1] == "c:cmd -c \"e|f\"") {
//t.Fatalf("failed to get part: %v", td)
//}
//}

func TestParseParts(t *testing.T) {
	s := "a b c d  eaa | a b c"
	a := parseParts(s)
	if !(len(a) == 2 &&
		a[0].Str == "a b c d  eaa " &&
		a[1].Str == " a b c") {
		for _, t := range a {
			fmt.Printf("%v\n", t)
		}
		t.Fatal()
	}
}
func TestParseParts2(t *testing.T) {
	s := "cmd1|cmd2\\|cmd3"
	a := parseParts(s)
	if !(len(a) == 2 &&
		a[0].Str == "cmd1" &&
		a[1].Str == "cmd2\\|cmd3") {

		for _, t := range a {
			fmt.Printf("%v\n", t)
		}
		t.Fatal()
	}
}
func TestParseParts3(t *testing.T) {
	s := "\"cmd1|cmd2\"a"
	a := parseParts(s)
	if !(len(a) == 1 && a[0].Str == "\"cmd1|cmd2\"a") {
		for _, t := range a {
			fmt.Printf("%v\n", t)
		}
		t.Fatal()
	}
}
func TestParseParts4(t *testing.T) {
	s := "\"cmd1|\\\"cmd2\"a"
	a := parseParts(s)
	if !(len(a) == 1 && a[0].Str == "\"cmd1|\\\"cmd2\"a") {
		for _, t := range a {
			fmt.Printf("%v\n", t)
		}
		t.Fatal()
	}
}
func TestParseParts5(t *testing.T) {
	s := "cmd1|\"cmd2\\|cmd3"
	a := parseParts(s)
	if !(len(a) == 2 &&
		a[0].Str == "cmd1" &&
		a[1].Str == "\"cmd2\\|cmd3") {

		for _, t := range a {
			fmt.Printf("%v\n", t)
		}
		t.Fatal()
	}
}
func TestParsePartsArgs0(t *testing.T) {
	s := "   a\\    eaa  \" h h h \"  "
	a := parseParts(s)
	if !(len(a) == 1 &&
		len(a[0].Args) == 3 &&
		a[0].Args[0].Str == "a\\ " &&
		a[0].Args[1].Str == "eaa" &&
		a[0].Args[2].Str == "\" h h h \"") {

		for _, t := range a {
			fmt.Printf("%v\n", t)
		}
		t.Fatal()
	}
}
func TestTokenTrim(t *testing.T) {
	tok := Token{Str: "a\\\\b"}
	if !(tok.Trim() == "a\\b") {
		t.Fatalf(tok.Trim())
	}
}
