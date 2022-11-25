package lrparser

import (
	"testing"

	"github.com/jmigpin/editor/util/testutil"
)

func TestVerticesData1(t *testing.T) {
	grammar := `
		^S = C C;
		C = "c" C | "d";
	`
	expect := `
		vertex0:
	        	[{0,^^^->^S},[$]]
	        	[{0,^S->[C C]},[$]]
	        	[{0,C->["c" C]},["c","d"]]
	        	[{0,C->"d"},["c","d"]]
	        	^S->vertex1
	        	C->vertex2
	        	"c"->vertex3
	        	"d"->vertex4
	        vertex1:
	        	[{1,^^^->^S},[$]]
	        vertex2:
	        	[{1,^S->[C C]},[$]]
	        	[{0,C->["c" C]},[$]]
	        	[{0,C->"d"},[$]]
	        	C->vertex6
	        	"c"->vertex7
	        	"d"->vertex8
	        vertex3:
	        	[{0,C->["c" C]},["c","d"]]
	        	[{1,C->["c" C]},["c","d"]]
	        	[{0,C->"d"},["c","d"]]
	        	C->vertex5
	        	"c"->vertex3
	        	"d"->vertex4
	        vertex4:
	        	[{1,C->"d"},["c","d"]]
	        vertex5:
	        	[{2,C->["c" C]},["c","d"]]
	        vertex6:
	        	[{2,^S->[C C]},[$]]
	        vertex7:
	        	[{0,C->["c" C]},[$]]
	        	[{1,C->["c" C]},[$]]
	        	[{0,C->"d"},[$]]
	        	C->vertex9
	        	"c"->vertex7
	        	"d"->vertex8
	        vertex8:
	        	[{1,C->"d"},[$]]
	        vertex9:
	        	[{2,C->["c" C]},[$]]
	`
	testRulesToVerticesMode1(t, grammar, expect)
}
func TestVerticesData2(t *testing.T) {
	grammar := `
		^E = E "+" T | T;
		T = T "*" F | F;
		F = "(" E ")" | "a";
	`
	expect := `
		vertex0:
	        	[{0,^^^->^E},[$]]
	        	[{0,^E->T},["+",$]]
	        	[{0,^E->[^E "+" T]},["+",$]]
	        	[{0,F->["(" ^E ")"]},["*","+",$]]
	        	[{0,F->"a"},["*","+",$]]
	        	[{0,T->F},["*","+",$]]
	        	[{0,T->[T "*" F]},["*","+",$]]
	        	^E->vertex1
	        	F->vertex2
	        	T->vertex3
	        	"("->vertex4
	        	"a"->vertex5
	        vertex1:
	        	[{1,^^^->^E},[$]]
	        	[{1,^E->[^E "+" T]},["+",$]]
	        	"+"->vertex20
	        vertex2:
	        	[{1,T->F},["*","+",$]]
	        vertex3:
	        	[{1,^E->T},["+",$]]
	        	[{1,T->[T "*" F]},["*","+",$]]
	        	"*"->vertex18
	        vertex4:
	        	[{0,^E->T},[")","+"]]
	        	[{0,^E->[^E "+" T]},[")","+"]]
	        	[{0,F->["(" ^E ")"]},[")","*","+"]]
	        	[{1,F->["(" ^E ")"]},["*","+",$]]
	        	[{0,F->"a"},[")","*","+"]]
	        	[{0,T->F},[")","*","+"]]
	        	[{0,T->[T "*" F]},[")","*","+"]]
	        	^E->vertex6
	        	F->vertex7
	        	T->vertex8
	        	"("->vertex9
	        	"a"->vertex10
	        vertex5:
	        	[{1,F->"a"},["*","+",$]]
	        vertex6:
	        	[{1,^E->[^E "+" T]},[")","+"]]
	        	[{2,F->["(" ^E ")"]},["*","+",$]]
	        	")"->vertex17
	        	"+"->vertex13
	        vertex7:
	        	[{1,T->F},[")","*","+"]]
	        vertex8:
	        	[{1,^E->T},[")","+"]]
	        	[{1,T->[T "*" F]},[")","*","+"]]
	        	"*"->vertex15
	        vertex9:
	        	[{0,^E->T},[")","+"]]
	        	[{0,^E->[^E "+" T]},[")","+"]]
	        	[{0,F->["(" ^E ")"]},[")","*","+"]]
	        	[{1,F->["(" ^E ")"]},[")","*","+"]]
	        	[{0,F->"a"},[")","*","+"]]
	        	[{0,T->F},[")","*","+"]]
	        	[{0,T->[T "*" F]},[")","*","+"]]
	        	^E->vertex11
	        	F->vertex7
	        	T->vertex8
	        	"("->vertex9
	        	"a"->vertex10
	        vertex10:
	        	[{1,F->"a"},[")","*","+"]]
	        vertex11:
	        	[{1,^E->[^E "+" T]},[")","+"]]
	        	[{2,F->["(" ^E ")"]},[")","*","+"]]
	        	")"->vertex12
	        	"+"->vertex13
	        vertex12:
	        	[{3,F->["(" ^E ")"]},[")","*","+"]]
	        vertex13:
	        	[{2,^E->[^E "+" T]},[")","+"]]
	        	[{0,F->["(" ^E ")"]},[")","*","+"]]
	        	[{0,F->"a"},[")","*","+"]]
	        	[{0,T->F},[")","*","+"]]
	        	[{0,T->[T "*" F]},[")","*","+"]]
	        	F->vertex7
	        	T->vertex14
	        	"("->vertex9
	        	"a"->vertex10
	        vertex14:
	        	[{3,^E->[^E "+" T]},[")","+"]]
	        	[{1,T->[T "*" F]},[")","*","+"]]
	        	"*"->vertex15
	        vertex15:
	        	[{0,F->["(" ^E ")"]},[")","*","+"]]
	        	[{0,F->"a"},[")","*","+"]]
	        	[{2,T->[T "*" F]},[")","*","+"]]
	        	F->vertex16
	        	"("->vertex9
	        	"a"->vertex10
	        vertex16:
	        	[{3,T->[T "*" F]},[")","*","+"]]
	        vertex17:
	        	[{3,F->["(" ^E ")"]},["*","+",$]]
	        vertex18:
	        	[{0,F->["(" ^E ")"]},["*","+",$]]
	        	[{0,F->"a"},["*","+",$]]
	        	[{2,T->[T "*" F]},["*","+",$]]
	        	F->vertex19
	        	"("->vertex4
	        	"a"->vertex5
	        vertex19:
	        	[{3,T->[T "*" F]},["*","+",$]]
	        vertex20:
	        	[{2,^E->[^E "+" T]},["+",$]]
	        	[{0,F->["(" ^E ")"]},["*","+",$]]
	        	[{0,F->"a"},["*","+",$]]
	        	[{0,T->F},["*","+",$]]
	        	[{0,T->[T "*" F]},["*","+",$]]
	        	F->vertex2
	        	T->vertex21
	        	"("->vertex4
	        	"a"->vertex5
	        vertex21:
	        	[{3,^E->[^E "+" T]},["+",$]]
	        	[{1,T->[T "*" F]},["*","+",$]]
	        	"*"->vertex18
	`
	testRulesToVerticesMode1(t, grammar, expect)
}
func TestVerticesData3(t *testing.T) {
	grammar := `		
		^id = id "a" | "a";
	`
	expect := `
		vertex0:
	        	[{0,^^^->^id},[$]]
	        	[{0,^id->[^id "a"]},["a",$]]
	        	[{0,^id->"a"},["a",$]]
	        	^id->vertex1
	        	"a"->vertex2
	        vertex1:
	        	[{1,^^^->^id},[$]]
	        	[{1,^id->[^id "a"]},["a",$]]
	        	"a"->vertex3
	        vertex2:
	        	[{1,^id->"a"},["a",$]]
	        vertex3:
	        	[{2,^id->[^id "a"]},["a",$]]
	`
	testRulesToVerticesMode1(t, grammar, expect)
}
func TestVerticesData4(t *testing.T) {
	grammar := `		
		^id = id "b" "a" | "a";
	`
	expect := `
		vertex0:
	        	[{0,^^^->^id},[$]]
	        	[{0,^id->[^id "b" "a"]},["b",$]]
	        	[{0,^id->"a"},["b",$]]
	        	^id->vertex1
	        	"a"->vertex2
	        vertex1:
	        	[{1,^^^->^id},[$]]
	        	[{1,^id->[^id "b" "a"]},["b",$]]
	        	"b"->vertex3
	        vertex2:
	        	[{1,^id->"a"},["b",$]]
	        vertex3:
	        	[{2,^id->[^id "b" "a"]},["b",$]]
	        	"a"->vertex4
	        vertex4:
	        	[{3,^id->[^id "b" "a"]},["b",$]]
	`
	testRulesToVerticesMode1(t, grammar, expect)
}
func TestVerticesData5(t *testing.T) {
	grammar := `		
		^id = "a" id | "a";		
	`
	expect := `
		vertex0:
	        	[{0,^^^->^id},[$]]
	        	[{0,^id->["a" ^id]},[$]]
	        	[{0,^id->"a"},[$]]
	        	^id->vertex1
	        	"a"->vertex2
	        vertex1:
	        	[{1,^^^->^id},[$]]
	        vertex2:
	        	[{0,^id->["a" ^id]},[$]]
	        	[{1,^id->["a" ^id]},[$]]
	        	[{0,^id->"a"},[$]]
	        	[{1,^id->"a"},[$]]
	        	^id->vertex3
	        	"a"->vertex2
	        vertex3:
	        	[{2,^id->["a" ^id]},[$]]
	`
	testRulesToVerticesMode1(t, grammar, expect)
}
func TestVerticesData6(t *testing.T) {
	grammar := `
		^id = "a" ("b")? "a";
	`
	expect := `
		vertex0:
	        	[{0,^^^->^id},[$]]
	        	[{0,^id->["a" ("b")? "a"]},[$]]
	        	^id->vertex1
	        	"a"->vertex2
	        vertex1:
	        	[{1,^^^->^id},[$]]
	        vertex2:
	        	[{1,^id->["a" ("b")? "a"]},[$]]
	        	[{0,("b")?->"b"},["a"]]
	        	[{1,("b")?->nil},["a"]]
	        	("b")?->vertex3
	        	"b"->vertex4
	        vertex3:
	        	[{2,^id->["a" ("b")? "a"]},[$]]
	        	"a"->vertex5
	        vertex4:
	        	[{1,("b")?->"b"},["a"]]
	        vertex5:
	        	[{3,^id->["a" ("b")? "a"]},[$]]
	`
	testRulesToVerticesMode1(t, grammar, expect)
}
func TestVerticesData7(t *testing.T) {
	grammar := `
		^id = "a" id | nil;
	`
	expect := `
		vertex0:
	        	[{0,^^^->^id},[$]]
	        	[{0,^id->["a" ^id]},[$]]
	        	[{1,^id->nil},[$]]
	        	^id->vertex1
	        	"a"->vertex2
	        vertex1:
	        	[{1,^^^->^id},[$]]
	        vertex2:
	        	[{0,^id->["a" ^id]},[$]]
	        	[{1,^id->["a" ^id]},[$]]
	        	[{1,^id->nil},[$]]
	        	^id->vertex3
	        	"a"->vertex2
	        vertex3:
	        	[{2,^id->["a" ^id]},[$]]
	`
	testRulesToVerticesMode1(t, grammar, expect)
}
func TestVerticesData8(t *testing.T) {
	grammar := `
		^S = "a" A "d" | "b" B "d" | "a" B "e" | "b" A "e";
		A = "c";
		B = "c";
	`
	expect := `
		vertex0:
	        	[{0,^^^->^S},[$]]
	        	[{0,^S->["a" A "d"]},[$]]
	        	[{0,^S->["a" B "e"]},[$]]
	        	[{0,^S->["b" A "e"]},[$]]
	        	[{0,^S->["b" B "d"]},[$]]
	        	^S->vertex1
	        	"a"->vertex2
	        	"b"->vertex3
	        vertex1:
	        	[{1,^^^->^S},[$]]
	        vertex2:
	        	[{1,^S->["a" A "d"]},[$]]
	        	[{1,^S->["a" B "e"]},[$]]
	        	[{0,A->"c"},["d"]]
	        	[{0,B->"c"},["e"]]
	        	A->vertex9
	        	B->vertex10
	        	"c"->vertex11
	        vertex3:
	        	[{1,^S->["b" A "e"]},[$]]
	        	[{1,^S->["b" B "d"]},[$]]
	        	[{0,A->"c"},["e"]]
	        	[{0,B->"c"},["d"]]
	        	A->vertex4
	        	B->vertex5
	        	"c"->vertex6
	        vertex4:
	        	[{2,^S->["b" A "e"]},[$]]
	        	"e"->vertex8
	        vertex5:
	        	[{2,^S->["b" B "d"]},[$]]
	        	"d"->vertex7
	        vertex6:
	        	[{1,A->"c"},["e"]]
	        	[{1,B->"c"},["d"]]
	        vertex7:
	        	[{3,^S->["b" B "d"]},[$]]
	        vertex8:
	        	[{3,^S->["b" A "e"]},[$]]
	        vertex9:
	        	[{2,^S->["a" A "d"]},[$]]
	        	"d"->vertex13
	        vertex10:
	        	[{2,^S->["a" B "e"]},[$]]
	        	"e"->vertex12
	        vertex11:
	        	[{1,A->"c"},["d"]]
	        	[{1,B->"c"},["e"]]
	        vertex12:
	        	[{3,^S->["a" B "e"]},[$]]
	        vertex13:
	        	[{3,^S->["a" A "d"]},[$]]
	`
	testRulesToVerticesMode1(t, grammar, expect)
}

//----------

//func TestVerticesData9(t *testing.T) {
//	grammar := `
//		^S = (s1)%
//		s1 = (letter digit)+
//	`
//	expect := ``
//	testRulesToVerticesMode1(t, grammar, expect)
//}
//func TestVerticesData9(t *testing.T) {
//	grammar := `
//		^S = "a" ("a"|"1")*

//		#^S = "a" s2
//		#s2 = "a" s2 | "1" s2 | nil
//	`
//	expect := `
//	`
//	testRulesToVerticesMode1(t, grammar, expect)
//}

//func TestVerticesData10(t *testing.T) {
//	// TODO: crash, should give loop error
//	// ^S = S
//	grammar := `
//		^S = S
//		#^S = S | "a"
//	`
//	expect := `
//	`
//	testRulesToVerticesMode1(t, grammar, expect)
//}

//----------
//----------
//----------

func testRulesToVerticesMode1(t *testing.T, grammar, expect string) {
	t.Helper()

	fset := &FileSet{Src: []byte(grammar), Filename: "_.grammar"}
	ri := newRuleIndex()
	if err := setupPredefineds(ri); err != nil {
		t.Fatal(err)
	}
	gp := newGrammarParser(ri)
	if err := gp.parse(fset); err != nil {
		t.Fatal(err)
	}

	vd, err := newVerticesData(ri, "", false)
	if err != nil {
		t.Fatal(err)
	}
	//t.Log(ri) // index deref'd

	res := vd.String()

	res2 := testutil.TrimLineSpaces(res)
	expect2 := testutil.TrimLineSpaces(expect)
	if res2 != expect2 {
		t.Fatal(res)
	}
}
