package toolbardata

import (
	"fmt"
	"testing"
)

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
func TestParseParts6(t *testing.T) {
	s := "|cmd1|cmd2"
	a := parseParts(s)
	if !(len(a) == 3 && len(a[0].Args) == 0) {
		for _, t := range a {
			fmt.Printf("%v\n", t)
		}
		t.Fatal()
	}
}
func TestParseParts7(t *testing.T) {
	s := ""
	a := parseParts(s)
	if !(len(a) == 0) {
		for _, t := range a {
			fmt.Printf("%v\n", t)
		}
		t.Fatal()
	}
}
func TestParseParts8(t *testing.T) {
	s := " "
	a := parseParts(s)
	if !(len(a) == 1) {
		t.Fatal()
	}
}
func TestParseParts9(t *testing.T) {
	s := "|"
	a := parseParts(s)
	if !(len(a) == 1) {
		t.Fatal()
	}
}
func TestParseParts10(t *testing.T) {
	s := "| "
	a := parseParts(s)
	if !(len(a) == 2) {
		t.Fatal()
	}
}
func TestParsePartsArgs0(t *testing.T) {
	s := "   a\\    eaa  \" h  h h \"  "
	a := parseParts(s)
	if !(len(a) == 1 &&
		len(a[0].Args) == 3 &&
		a[0].Args[0].Str == "a\\ " &&
		a[0].Args[1].Str == "eaa" &&
		a[0].Args[2].Str == "\" h  h h \"") {

		for _, t := range a {
			for _, t2 := range t.Args {
				fmt.Printf("%v\n", t2)
			}
		}
		t.Fatal()
	}
}
