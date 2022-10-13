package lrparser

import (
	"fmt"
	"testing"

	"github.com/jmigpin/editor/util/testutil"
)

func TestGrammarParser1(t *testing.T) {
	in := `
		rule ^S = C C
		rule C = "c" C | "d"
	`
	out := `
		ruleindex:
        	^S = {ref:C} {ref:C}
	        C = "c" {ref:C} | "d"
	`
	testGrammarParserMode1(t, in, out)
}
func TestGrammarParser2(t *testing.T) {
	in := `
		rule ^S = (a|b|"cd"&)?
	`
	out := `
		ruleindex:
		^S = ({ref:a} | {ref:b} | "cd"&)?
	`
	testGrammarParserMode1(t, in, out)
}

//----------

func testGrammarParserMode1(t *testing.T, in, out string) {
	t.Helper()
	fset := &FileSet{Src: []byte(in), Filename: "_.grammar"}
	gp := newGrammarParser()
	ri, err := gp.parse(fset)
	if err != nil {
		t.Fatal(err)
	}
	//t.Logf("\n%v\n", ri)
	res := fmt.Sprintf("%v", ri)
	res2 := testutil.TrimLineSpaces(res)
	expect2 := testutil.TrimLineSpaces(out)
	if res2 != expect2 {
		t.Fatal(res)
	}
}
