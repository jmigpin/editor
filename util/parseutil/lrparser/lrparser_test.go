package lrparser

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/jmigpin/editor/util/astut"
	"github.com/jmigpin/editor/util/testutil"
	"golang.org/x/tools/go/ast/astutil"
)

func TestLrparser1(t *testing.T) {
	gram := `
		^S = C C;
		C = "c" C | "d";
	`
	in := "●ccdd"
	out := `
		-> ^S: "ccdd"
			-> C: "ccd"
				-> "c": "c"
				-> C: "cd"
					-> "c": "c"
					-> C: "d"
						-> "d": "d"
			-> C: "d"
				-> "d": "d"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser2(t *testing.T) {
	gram := `
		^id = "a" id | "a";
	`
	in := "●aaa"
	out := `
		-> ^id: "aaa"
			-> "a": "a"
			-> ^id: "aa"
				-> "a": "a"
				-> ^id: "a"
					-> "a": "a"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser3(t *testing.T) {
	gram := `
		^id = id "a" | "a";
	`
	in := "●aaa"
	out := `
		-> ^id: "aaa"
			-> ^id: "aa"
				-> ^id: "a"
					-> "a": "a"
				-> "a": "a"
			-> "a": "a"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser4(t *testing.T) {
	gram := `
		^id = (digit)?;
	`
	in := "●1"
	out := `
		-> ^id: "1"
			-> (digit)?: "1"
				-> digit: "1"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser5(t *testing.T) {
	gram := `
		#^id = letter id2 letter;
		#id2 = digit | nil;
		^id = letter (digit)? letter;	
	`
	in := "●aa"
	out := `
		-> ^id: "aa"
			-> letter: "a"
			-> (digit)?: ""
			-> letter: "a"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser6(t *testing.T) {
	gram := `		
		^id = letter (letter|digit)* digit;
	`
	in := "●a11"
	out := `
		-> ^id: "a11"
			-> letter: "a"
			-> ([letter|digit])*: "1"
				-> ([letter|digit])*: ""
				-> [letter|digit]: "1"
					-> digit: "1"
			-> digit: "1"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser7(t *testing.T) {
	gram := `
		^id = (letter|digit)*;
	`
	in := "●a1"
	out := `
		-> ^id: "a1"
			-> ([letter|digit])*: "a1"
				-> ([letter|digit])*: "a"
					-> ([letter|digit])*: ""
					-> [letter|digit]: "a"
						-> letter: "a"
				-> [letter|digit]: "1"
					-> digit: "1"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser7b(t *testing.T) {
	gram := `
		^id = (letter|digit)+;
	`
	in := "●a1"
	out := `
		-> ^id: "a1"
			-> ([letter|digit])+: "a1"
				-> ([letter|digit])*: "a"
					-> ([letter|digit])*: ""
					-> [letter|digit]: "a"
						-> letter: "a"
				-> [letter|digit]: "1"
					-> digit: "1"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser8(t *testing.T) {
	gram := `
		#^S = "a" ("a"|"1")*;
		
		^S = "a" s2;
		s2 = "a" s2 | "1" s2 | nil;
	`
	in := "●aa1"
	out := `
		-> ^S: "aa1"
			-> "a": "a"
			-> s2: "a1"
				-> "a": "a"
				-> s2: "1"
					-> "1": "1"
					-> s2: ""
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser9a(t *testing.T) {
	gram := `
		^S = letter (letter)*;
	`
	in := "●aaa"
	out := `
		-> ^S: "aaa"
			-> letter: "a"
			-> (letter)*: "aa"
				-> (letter)*: "a"
					-> (letter)*: ""
					-> letter: "a"
				-> letter: "a"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser9b(t *testing.T) {
	gram := `
		^S = letter (letter)*;
	`
	in := "●a"
	out := `
		-> ^S: "a"
			-> letter: "a"
			-> (letter)*: ""
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser9c(t *testing.T) {
	gram := `
		^S = letter (letter)+;
	`
	in := "●aaaa"
	out := `
		-> ^S: "aaaa"
			-> letter: "a"
			-> (letter)+: "aaa"
				-> (letter)*: "aa"
					-> (letter)*: "a"
						-> (letter)*: ""
						-> letter: "a"
					-> letter: "a"
				-> letter: "a"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser10(t *testing.T) {
	gram := `
		^S = (letter|digit)?;
	`
	in := "●1"
	out := `
		-> ^S: "1"
			-> ([letter|digit])?: "1"
				-> [letter|digit]: "1"
					-> digit: "1"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser11(t *testing.T) {
	gram := `
		^S = (letter digit)+;
	`
	in := "●a1b2c3"
	out := `
		-> ^S: "a1b2c3"
			-> ([letter digit])+: "a1b2c3"
				-> ([letter digit])*: "a1b2"
					-> ([letter digit])*: "a1"
						-> ([letter digit])*: ""
						-> [letter digit]: "a1"
							-> letter: "a"
							-> digit: "1"
					-> [letter digit]: "b2"
						-> letter: "b"
						-> digit: "2"
				-> [letter digit]: "c3"
					-> letter: "c"
					-> digit: "3"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser11b(t *testing.T) {
	gram := `
		^S = (letter digit)*;
	`
	in := "●a1b2c3"
	out := `
		-> ^S: "a1b2c3"
			-> ([letter digit])*: "a1b2c3"
				-> ([letter digit])*: "a1b2"
					-> ([letter digit])*: "a1"
						-> ([letter digit])*: ""
						-> [letter digit]: "a1"
							-> letter: "a"
							-> digit: "1"
					-> [letter digit]: "b2"
						-> letter: "b"
						-> digit: "2"
				-> [letter digit]: "c3"
					-> letter: "c"
					-> digit: "3"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser11c(t *testing.T) {
	gram := `
		^S = letter digit;
	`
	in := "●a1"
	out := `
		-> ^S: "a1"
			-> letter: "a"
			-> digit: "1"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser12(t *testing.T) {
	gram := `
		^S = ((":\"'")%)+;
	`
	in := "●:\":"
	out := `
		-> ^S: ":\":"
	        	-> (":\"'"%)+: ":\":"
	        		-> (":\"'"%)*: ":\""
	        			-> (":\"'"%)*: ":"
	        				-> (":\"'"%)*: ""
	        				-> ":\"'"%: ":"
	        			-> ":\"'"%: "\""
	        		-> ":\"'"%: ":"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser13(t *testing.T) {
	gram := `
		^S = letter (letter|digit)* digit;
	`
	in := "●aa11"
	out := `
		-> ^S: "aa11"
			-> letter: "a"
			-> ([letter|digit])*: "a1"
				-> ([letter|digit])*: "a"
					-> ([letter|digit])*: ""
					-> [letter|digit]: "a"
						-> letter: "a"
				-> [letter|digit]: "1"
					-> digit: "1"
			-> digit: "1"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser14(t *testing.T) {
	gram := `
		// reduce/reduce conflict
		^S = (s2)+; 
		s2 = (letter)+;		
	`
	in := "●aa"
	out := `
	`
	// shift/reduce conflict: either make
	// - several s2, each with one letter
	// - or one s2 with many letters
	//testLrparserMode1(t, gram, in, out) // conflict

	// use shift by default
	_, err := testLrparserMode3(t, gram, in, out, false, false, true)
	if err == nil {
		t.Fatal("expecting error")
	}
	if !strings.Contains(err.Error(), "conflict") {
		t.Fatal("expecting conflict error")
	}
}
func TestLrparser15(t *testing.T) {
	gram := `
		^S = (s2)~ s3;
		s2 = "abc";
		s3 = "de";
	`
	in := "ab●cde"
	out := `
		-> ^S: "cde"
			-> "abc"~: "c"
			-> s3: "de"
				-> "de": "de"
`
	testLrparserMode2(t, gram, in, out, false, false, false)
}
func TestLrparser16(t *testing.T) {
	gram := `
		^S = (s2)~ ((s2)%)+;
		s2 = "abc";
	`
	in := "ab●caa"
	out := `
		-> ^S: "caa"
	        	-> "abc"~: "c"
	        	-> ("abc"%)+: "aa"
	        		-> ("abc"%)*: "a"
	        			-> ("abc"%)*: ""
	        			-> "abc"%: "a"
	        		-> "abc"%: "a"
`
	testLrparserMode2(t, gram, in, out, false, false, false)
}
func TestLrparser17(t *testing.T) {
	gram := `
		^S = ("+")! "+";
	`
	in := "●0+"
	out := `
		-> ^S: "0+"
	        	-> "+"!: "0"
	        	-> "+": "+"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser18(t *testing.T) {
	gram := `
		^S = (s2)! "+";
		s2 = ("abc")%|("defg")%;
		//s2 = ("abc"|"defg")%;
	`
	in := "●h+"
	out := `
		-> ^S: "h+"
	        	-> "abcdefg"!: "h"
	        	-> "+": "+"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser19(t *testing.T) {
	gram := `
		^S = (@dropRunes((s2)%,("b")%))+;
		s2 = "abc";
	`
	in := "●ac"
	out := `
		-> ^S: "ac"
	        	-> ("ac"%)+: "ac"
	        		-> ("ac"%)*: "a"
	        			-> ("ac"%)*: ""
	        			-> "ac"%: "a"
	        		-> "ac"%: "c"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser20(t *testing.T) {
	gram := `
		^S = (s2|s3|s4)+;
		s2 = "ab";
		s3 = "c";
		s4 = "d";
	`
	in := "●ababdabcd"
	out := `
		-> ^S: "ababdabcd"
			-> ([s2|s3|s4])+: "ababdabcd"
				-> ([s2|s3|s4])*: "ababdabc"
					-> ([s2|s3|s4])*: "ababdab"
						-> ([s2|s3|s4])*: "ababd"
							-> ([s2|s3|s4])*: "abab"
								-> ([s2|s3|s4])*: "ab"
									-> ([s2|s3|s4])*: ""
									-> [s2|s3|s4]: "ab"
										-> s2: "ab"
											-> "ab": "ab"
								-> [s2|s3|s4]: "ab"
									-> s2: "ab"
										-> "ab": "ab"
							-> [s2|s3|s4]: "d"
								-> s4: "d"
									-> "d": "d"
						-> [s2|s3|s4]: "ab"
							-> s2: "ab"
								-> "ab": "ab"
					-> [s2|s3|s4]: "c"
						-> s3: "c"
							-> "c": "c"
				-> [s2|s3|s4]: "d"
					-> s4: "d"
						-> "d": "d"
`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser21(t *testing.T) {
	gram := `
		//^S = (sep)* arg args2;		
		//args2 = (sep)+ arg args2 | (sep)+ | nil; // ok
		//args2 = (sep)+ arg args2 | (sep)*; // ok (was conflict)		
		
		^S = (sep)* arg ((sep)+ arg)* (sep)*; // ok
		
		//^S = (sep)* arg ((sep)+ arg (sep)*)* ;
		//^S = ((sep)* arg)+ (sep)*;
		sep = " ";
		arg = "a";
	`
	in := "●  a  a  "
	out := `
		-> ^S: "  a  a  "
			-> (sep)*: "  "
				-> (sep)*: " "
					-> (sep)*: ""
					-> sep: " "
						-> " ": " "
				-> sep: " "
					-> " ": " "
			-> arg: "a"
				-> "a": "a"
			-> ([(sep)+ arg])*: "  a"
				-> ([(sep)+ arg])*: ""
				-> [(sep)+ arg]: "  a"
					-> (sep)+: "  "
						-> (sep)*: " "
							-> (sep)*: ""
							-> sep: " "
								-> " ": " "
						-> sep: " "
							-> " ": " "
					-> arg: "a"
						-> "a": "a"
			-> (sep)*: "  "
				-> (sep)*: " "
					-> (sep)*: ""
					-> sep: " "
						-> " ": " "
				-> sep: " "
					-> " ": " "
`
	testLrparserMode1(t, gram, in, out)
}

//func TestLrparser22(t *testing.T) {
//	gram := `
//		^S = ((letter)!)+;
//	`
//	in := "●a "
//	out := `
//`
//	testLrparserMode1(t, gram, in, out)
//}

//----------

func TestLrparserStop1(t *testing.T) {
	gram := `
		^id =  digit (letter)*;		
	`
	in := "<<<●1ab>>>"
	out := `
		-> ^id: "1ab"
			-> digit: "1"
			-> (letter)*: "ab"
				-> (letter)*: "a"
					-> (letter)*: ""
					-> letter: "a"
				-> letter: "b"
`
	testLrparserMode2(t, gram, in, out, false, true, false)
}
func TestLrparserStop2(t *testing.T) {
	gram := `
		^S = letter (linecol)?;
		linecol = entry (entry)?;
		entry = ":" (digit)+;	
	`
	in := "●a:1:++"
	out := `
		-> ^S: "a:1"
			-> letter: "a"
			-> (linecol)?: ":1"
				-> linecol: ":1"
					-> entry: ":1"
						-> ":": ":"
						-> (digit)+: "1"
							-> (digit)*: ""
							-> digit: "1"
					-> (entry)?: ""
`

	testLrparserMode2(t, gram, in, out, false, true, false)
}
func TestLrparserStop3(t *testing.T) {
	gram := `
		^S = (letter ":")+;
	`
	in := "●a:b:c:d"
	out := `
		-> ^S: "a:b:c:"
			-> ([letter ":"])+: "a:b:c:"
				-> ([letter ":"])*: "a:b:"
					-> ([letter ":"])*: "a:"
						-> ([letter ":"])*: ""
						-> [letter ":"]: "a:"
							-> letter: "a"
							-> ":": ":"
					-> [letter ":"]: "b:"
						-> letter: "b"
						-> ":": ":"
				-> [letter ":"]: "c:"
					-> letter: "c"
					-> ":": ":"
`

	testLrparserMode2(t, gram, in, out, false, true, false)
}
func TestLrparserStop3b(t *testing.T) {
	gram := `
		^S = (letter ":")+ (digit)?;
	`
	in := "●a:a:b"
	out := `
		-> ^S: "a:a:"
			-> ([letter ":"])+: "a:a:"
				-> ([letter ":"])*: "a:"
					-> ([letter ":"])*: ""
					-> [letter ":"]: "a:"
						-> letter: "a"
						-> ":": ":"
				-> [letter ":"]: "a:"
					-> letter: "a"
					-> ":": ":"
			-> (digit)?: ""
`

	testLrparserMode2(t, gram, in, out, false, true, false)
}
func TestLrparserStop4(t *testing.T) {
	gram := `
		^S = (letter digit letter)+;	
	`
	in := "●a1ab2bc3"
	out := `
		-> ^S: "a1ab2b"
			-> ([letter digit letter])+: "a1ab2b"
				-> ([letter digit letter])*: "a1a"
					-> ([letter digit letter])*: ""
					-> [letter digit letter]: "a1a"
						-> letter: "a"
						-> digit: "1"
						-> letter: "a"
				-> [letter digit letter]: "b2b"
					-> letter: "b"
					-> digit: "2"
					-> letter: "b"
`

	testLrparserMode2(t, gram, in, out, false, true, false)
}
func TestLrparserStop5(t *testing.T) {
	gram := `
		^S = ("a" "b" "c" "d")+;		
	`
	in := "●abcdab"
	out := `
		-> ^S: "abcd"
			-> (["a" "b" "c" "d"])+: "abcd"
				-> (["a" "b" "c" "d"])*: ""
				-> ["a" "b" "c" "d"]: "abcd"
					-> "a": "a"
					-> "b": "b"
					-> "c": "c"
					-> "d": "d"
`

	testLrparserMode2(t, gram, in, out, false, true, false)
}

//----------

func TestLrparserRev1(t *testing.T) {
	gram := `
		^rev =  digit (letter)*	;	
		#^rev =  (letter)* digit;
	`
	in := "<<<1ab●>>>"
	//in := "<<<●1ab>>>"
	out := `
		-> ^rev: "1ab"
			-> digit: "1"
			-> (letter)*: "ab"
				-> (letter)*: "b"
					-> (letter)*: ""
					-> letter: "b"
				-> letter: "a"
`
	testLrparserMode2(t, gram, in, out, true, true, false)
	//testLrparserMode2(t, gram, in, out, false, true, false) // no rev
}
func TestLrparserRev2(t *testing.T) {
	gram := `
		^S = (s2)?;
		s2 = (letter)+;
	`
	in := "aa●11"
	out := `
		-> ^S: "aa"
			-> (s2)?: "aa"
				-> s2: "aa"
					-> (letter)+: "aa"
						-> (letter)*: "a"
							-> (letter)*: ""
							-> letter: "a"
						-> letter: "a"
`
	//testLrparserMode2(t, gram, in, out, true, true)
	testLrparserMode2(t, gram, in, out, true, false, false)
}
func TestLrparserRev3(t *testing.T) {
	gram := `
		^S = (letter|esc)+;
		esc = @escapeAny(0,"\\"); 
	`
	in := "a b\\ c●"
	out := `
		-> ^S: "b\\ c"
	        	-> ([letter|esc])+: "b\\ c"
	        		-> ([letter|esc])*: "\\ c"
	        			-> ([letter|esc])*: "c"
	        				-> ([letter|esc])*: ""
	        				-> [letter|esc]: "c"
	        					-> letter: "c"
	        			-> [letter|esc]: "\\ "
	        				-> esc: "\\ "
	        					-> escapeAny('\\'): "\\ "
	        		-> [letter|esc]: "b"
	        			-> letter: "b"
`

	testLrparserMode2(t, gram, in, out, true, true, false)
}
func TestLrparserRev4(t *testing.T) {
	gram := `
		^S = (letter|esc)*;
		esc = @escapeAny(0,"\\"); 
	`
	in := "aaa ●bbb"
	out := `
		-> ^S: ""
			-> ([letter|esc])*: ""
`

	bnd := testLrparserMode2(t, gram, in, out, true, true, false)
	if bnd.Pos() != 4 {
		t.Fatalf("bad pos: %v", bnd.Pos())
	}
}
func TestLrparserRev5(t *testing.T) {
	gram := `
		^S = (@escapeAny(0,esc))+ (esc)+;
		esc = "\\";	
	`
	in := "aaa\\:\\ \\● bb"
	out := `
		-> ^S: "\\:\\ \\"
	        	-> (escapeAny('\\'))+: "\\:\\ "
	        		-> (escapeAny('\\'))*: "\\ "
	        			-> (escapeAny('\\'))*: ""
	        			-> escapeAny('\\'): "\\ "
	        		-> escapeAny('\\'): "\\:"
	        	-> (esc)+: "\\"
	        		-> (esc)*: ""
	        		-> esc: "\\"
	        			-> "\\": "\\"
`
	bnd := testLrparserMode2(t, gram, in, out, true, true, false)
	if bnd.End() != 3 {
		t.Fatalf("bad pos: %v", bnd.End())
	}
}

//----------

func TestLrparserErr1(t *testing.T) {
	gram := `
		^S = (letter)? letter;	
	`
	in := "●a"
	out := ``
	_, err := testLrparserMode3(t, gram, in, out, false, true, true)
	t.Log(err)
	if err == nil {
		t.Fatal("expecting error")
	}
}
func TestLrparserErr2(t *testing.T) {
	gram := `
		^S = (digit)+ (letter)? digit; // was endless loop "●111+a"
	`
	in := "●111+a"
	out := ``
	_, err := testLrparserMode3(t, gram, in, out, false, true, true)
	t.Log(err)
	if err == nil {
		t.Fatal("expecting error")
	}
}
func TestLrparserErr3(t *testing.T) {
	gram := `
		^S = (digit)+ (letter)? ":";
		//^S = (digit)+ (letter)?; // ok
	`
	in := "●111a"
	out := ``
	_, err := testLrparserMode3(t, gram, in, out, false, true, true)
	t.Log(err)
	if err == nil {
		t.Fatal("expecting error")
	}
}
func TestLrparserErr4(t *testing.T) {
	gram := `
		^S = ((letter)!)%;
	`
	in := "●aa"
	out := ``
	_, err := testLrparserMode3(t, gram, in, out, false, true, true)
	t.Log(err)
	if err == nil {
		t.Fatal("expecting error")
	}
}

//----------

func TestLrparserBuild1(t *testing.T) {
	gram := `
		^S = (sep)* arg ((sep)+ arg)* (sep)*;
		arg = (letter|digit)+;
		sep = "-";
	`
	cp := testLrparserModeB(t, gram, false, true, true)

	args := []string{}
	cp.SetBuildNodeFn("S", func(d *BuildNodeData) error {
		//d.PrintRuleTree(5)
		if err := d.ChildLoop(2, func(d2 *BuildNodeData) error {
			//d2.PrintRuleTree(5)
			args = append(args, d2.ChildStr(1))
			return nil
		}); err != nil {
			return err
		}
		return nil
	})

	in := "●a1--b2-c3---d4--"
	bnd, _, err := testLrparserModeB2(t, cp, in)
	if err != nil {
		t.Fatal(err)
	}
	_ = bnd

	r1 := fmt.Sprintf("%v", args)
	if r1 != "[b2 c3 d4]" {
		t.Fatal(r1)
	}
}

func TestLrparserBuild2(t *testing.T) {
	gram := `
		^S = args;
		args = (sep)+ arg args | nil;
		arg = (letter|digit)+;
		sep = "-";
	`
	cp := testLrparserModeB(t, gram, false, true, true)

	args := []string{}
	cp.SetBuildNodeFn("S", func(d *BuildNodeData) error {
		//d.PrintRuleTree(5)
		if err := d.ChildLoop2(0, 2, func(d2 *BuildNodeData) error {
			// d2 is "args" rule
			if d2.ChildsLen() == 3 {
				args = append(args, d2.ChildStr(1))
			}
			return nil
		}, nil); err != nil {
			return err
		}
		return nil
	})

	in := "●-a1-b2-c3--d4"
	bnd, _, err := testLrparserModeB2(t, cp, in)
	if err != nil {
		t.Fatal(err)
	}
	_ = bnd

	r1 := fmt.Sprintf("%v", args)
	if r1 != "[a1 b2 c3 d4]" {
		t.Fatal(r1)
	}
}

//----------
//----------
//----------

func testLrparserMode1(t *testing.T, gram, in, out string) *BuildNodeData {
	t.Helper()
	return testLrparserMode2(t, gram, in, out, false, false, false)
}
func testLrparserMode2(t *testing.T, gram, in, out string, reverse, earlyStop, shiftOnSRConflict bool) *BuildNodeData {
	t.Helper()
	bnd, err := testLrparserMode3(t, gram, in, out, reverse, earlyStop, shiftOnSRConflict)
	if err != nil {
		t.Fatal(err)
	}
	return bnd
}
func testLrparserMode3(t *testing.T, gram, in, out string, reverse, earlyStop, shiftOnSRConflict bool) (*BuildNodeData, error) {
	t.Helper()

	//in = string(bytes.TrimRight(in, "\n"))
	//out = string(bytes.TrimRight(out, "\n"))

	in2, index, err := testutil.SourceCursor("●", string(in), 0)
	if err != nil {
		return nil, err
	}

	lrp, err := NewLrparserFromString(gram)
	if err != nil {
		return nil, err
	}

	opt := &CpOpt{
		StartRule:         "",
		Reverse:           reverse,
		EarlyStop:         earlyStop,
		ShiftOnSRConflict: shiftOnSRConflict,
		VerboseError:      true,
	}
	cp, err := lrp.ContentParser(opt)
	if err != nil {
		return nil, err
	}

	bnd, cpr, err := cp.Parse([]byte(in2), index)
	if err != nil {
		return nil, err
	}
	res := bnd.SprintRuleTree(-1)

	if *updateOutputFlag {
		fmt.Printf("%v\n", sprintTaggedOut(res))
	}

	res2 := testutil.TrimLineSpaces(res)
	expect2 := testutil.TrimLineSpaces(out)
	if res2 != expect2 {
		_ = cpr
		return nil, fmt.Errorf("%v\n%s", res, cpr.Debug(cp))
		//return nil, fmt.Errorf("%v", res)
	}
	return bnd, nil
}

//----------

func testLrparserModeB(t *testing.T, gram string, reverse, earlyStop, shiftOnSRConflict bool) *ContentParser {
	t.Helper()

	lrp, err := NewLrparserFromString(gram)
	if err != nil {
		t.Fatal(err)
	}

	opt := &CpOpt{
		StartRule:         "",
		Reverse:           reverse,
		EarlyStop:         earlyStop,
		ShiftOnSRConflict: shiftOnSRConflict,
		VerboseError:      true,
	}
	cp, err := lrp.ContentParser(opt)
	if err != nil {
		t.Fatal(err)
	}
	return cp
}
func testLrparserModeB2(t *testing.T, cp *ContentParser, in string) (*BuildNodeData, *cpRun, error) {
	in2, index, err := testutil.SourceCursor("●", string(in), 0)
	if err != nil {
		t.Fatal(err)
	}
	return cp.Parse([]byte(in2), index)
}

//----------
//----------
//----------

// WARNING: used for rewriting this file (yes, writes itself) with updated tests output when changing output formats
func TestUpdateOutput(t *testing.T) {
	return // MANUALLY DISABLED

	if !*updateOutputFlag {
		return
	}
	if err := updateOutput(t); err != nil {
		t.Fatal(err)
	}
}
func updateOutput(tt *testing.T) error {
	// configuration
	filename := "lrparser_test.go"
	//testNamePrefix := "TestLrparser"
	//testNamePrefix := "TestLrparserStop"
	testNamePrefix := "TestLrparserRev"
	contentVarName := "out"

	src, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	mode := parser.Mode(parser.ParseComments)
	astFile, err := parser.ParseFile(fset, filename, src, mode)
	if err != nil {
		return err
	}

	vis := (func(c *astutil.Cursor) bool)(nil)
	replace := (func(c *astutil.Cursor, val string) bool)(nil)

	// find test and run it
	vis = func(c *astutil.Cursor) bool {
		switch t := c.Node().(type) {
		case *ast.FuncDecl:
			funcName := t.Name.Name
			if strings.HasPrefix(funcName, testNamePrefix) {
				// run test
				funcName += "$" // ensure match
				fmt.Printf("running: %v\n", funcName)
				cmd := exec.Command("go", "test", fmt.Sprintf("-run=%v", funcName), "-update")
				out, err := cmd.Output()
				if err != nil {
					_ = err
					//tt.Fatal(err)
					//break
				}
				// parse wanted output
				out2, ok := parseTaggedOut(string(out))
				if !ok {
					//tt.Fatalf("unable to parse tagged output:\n%q", out2)
					//fmt.Printf("no tagged output:\n%v", out2)
					fmt.Printf("\tno tagged output\n")
					break
				}
				out3 := "`\n" + indentStr("\t\t", out2) + "`"
				//fmt.Printf("out: %s\n", out3)

				// replace var content
				vis3 := func(c *astutil.Cursor) bool {
					if ok := replace(c, out3); ok {
						fmt.Printf("\treplaced\n")
						return false
					}
					return true
				}
				_ = astutil.Apply(c.Node(), vis3, nil)
			}
		}
		return true
	}
	// replace var content
	replace = func(c *astutil.Cursor, val string) bool {
		switch t := c.Node().(type) {
		case *ast.AssignStmt:
			if len(t.Lhs) >= 1 {
				if id, ok := t.Lhs[0].(*ast.Ident); ok {
					if id.Name == contentVarName {
						//fmt.Printf("***%v\n", id)
						if len(t.Rhs) == 1 {
							bl := &ast.BasicLit{
								Kind:  token.STRING,
								Value: val,
							}
							t.Rhs[0] = bl
							return true
						}
					}
				}
			}
		}
		return false
	}

	node := astutil.Apply(astFile, vis, nil)

	src2, err := astut.SprintNode2(fset, node)
	if err != nil {
		return err
	}
	//fmt.Println(src2)
	//filename += "AA" // TESTING
	if err := os.WriteFile(filename, []byte(src2), 0o644); err != nil {
		return err
	}

	return nil
}

//----------
//----------
//----------

var updateOutputFlag = flag.Bool("update", false, "")

var parseOutTag1 = "=[output:start]="
var parseOutTag2 = "=[output:end]="

func sprintTaggedOut(s string) string {
	return fmt.Sprintf("%v\n%v\n%v", parseOutTag1, s, parseOutTag2)
}
func parseTaggedOut(s string) (string, bool) {
	i1 := strings.Index(s, parseOutTag1)
	i2 := strings.Index(s, parseOutTag2)
	if i1 < 0 || i2 < 0 {
		return s, false
	}
	return s[i1+len(parseOutTag1)+1 : i2-1], true // +1-1 for the newlines
}
