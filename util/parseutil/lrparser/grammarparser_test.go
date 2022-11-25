package lrparser

import (
	"fmt"
	"testing"

	"github.com/jmigpin/editor/util/testutil"
)

func TestGrammarParser1(t *testing.T) {
	in := `
		^S = C C;
		C = "c" C | "d";
	`
	out := `
		ruleindex{
			^S = [{r:C} {r:C}]
	        	C = [["c" {r:C}]|"d"]
	       	}
	`
	testGrammarParserMode1(t, in, out)
}
func TestGrammarParser2(t *testing.T) {
	in := `	
		^S = (a|b|("cd")%)?;
	`
	out := `
		ruleindex{
			^S = ([{r:a}|{r:b}|("cd")%])?
		}
	`
	testGrammarParserMode1(t, in, out)
}
func TestGrammarParser3(t *testing.T) {
	in := `
		^S = if a?b:c;
	`
	out := `
		ruleindex{
			^S = {if {r:a} ? {r:b} : {r:c}}
		}
	`
	testGrammarParserMode1(t, in, out)
}

//----------

func testGrammarParserMode1(t *testing.T, in, out string) {
	t.Helper()

	fset := &FileSet{Src: []byte(in), Filename: "_.grammar"}
	ri := newRuleIndex()
	gp := newGrammarParser(ri)
	if err := gp.parse(fset); err != nil {
		t.Fatal(err)
	}
	//t.Logf("\n%v\n", ri)

	res := fmt.Sprintf("%v", ri)
	res2 := testutil.TrimLineSpaces(res)
	expect2 := testutil.TrimLineSpaces(out)
	if res2 != expect2 {
		t.Fatal("\n" + res)
	}
}
