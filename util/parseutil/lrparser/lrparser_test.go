package lrparser

import (
	"fmt"
	"testing"

	"github.com/jmigpin/editor/util/testutil"
)

func TestLrparser1(t *testing.T) {
	gram := `
		^S = C C .
		C = "c" C | "d" .
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
		^id = "a" id | "a" .
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
		^id = id "a" | "a" .
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
		^id = (digit)? .
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
		#^id = letter id2 letter .
		#id2 = digit | nil .
		^id = letter (digit)? letter .	
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
		# conflict
		#^id = letter (letter|digit)* digit	 .	
		
		# conflict
		#^id = letter id2 digit .
		#id2 = letter id2 | digit id2 | nil .
		
		# ok
		^id = letter id2 .
		id2 = letter id2 | digit id2 | digit .
	`
	in := "●a11"
	out := `		
	         -> ^id: "a11"
	        	-> letter: "a"
	        	-> id2: "11"
	        		-> digit: "1"
	        		-> id2: "1"
	        			-> digit: "1"
        `
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser7(t *testing.T) {
	gram := `
		^id = (letter|digit)* .
	`
	in := "●a1"
	out := `
		-> ^id: "a1"
	        	-> ([letter | digit])*: "a1"
	        		-> [letter | digit]: "a"
	        			-> letter: "a"
	        		-> [letter | digit]: "1"
	        			-> digit: "1"
        `
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser7b(t *testing.T) {
	gram := `
		^id = (letter|digit)+ .
	`
	in := "●a1"
	out := `
		-> ^id: "a1"
	        	-> ([letter | digit])+: "a1"
	        		-> [letter | digit]: "a"
	        			-> letter: "a"
	        		-> [letter | digit]: "1"
	        			-> digit: "1"
        `
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser8(t *testing.T) {
	gram := `
		#^S = "a" ("a"|"1")* .
		
		^S = "a" s2 .
		s2 = "a" s2 | "1" s2 | nil .
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
		^S = letter (letter)* .
	`
	in := "●aaa"
	out := `
		-> ^S: "aaa"
	        	-> letter: "a"
	        	-> (letter)*: "aa"
	        		-> letter: "a"
	        		-> letter: "a"
	`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser9b(t *testing.T) {
	gram := `
		^S = letter (letter)* .
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
		^S = letter (letter)+ .
	`
	in := "●aaaa"
	out := `
		-> ^S: "aaaa"
	        	-> letter: "a"
	        	-> (letter)+: "aaa"
	        		-> letter: "a"
	        		-> letter: "a"
	        		-> letter: "a"
	`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser10(t *testing.T) {
	gram := `
		^S = (letter|digit)? .
	`
	in := "●1"
	out := `
		-> ^S: "1"
	        	-> ([letter | digit])?: "1"
	        		-> [letter | digit]: "1"
	        			-> digit: "1"
	`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser11(t *testing.T) {
	gram := `
		^S = (letter digit)+ .
	`
	in := "●a1b2c3"
	out := `
		-> ^S: "a1b2c3"
	        	-> ([letter digit])+: "a1b2c3"
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
		^S = (letter digit)* .
	`
	in := "●a1b2c3"
	out := `
		-> ^S: "a1b2c3"
	        	-> ([letter digit])*: "a1b2c3"
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
		^S = letter digit .
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
		^S = (":\"'"&)+ .
	`
	in := "●:\":"
	out := `
		-> ^S: ":\":"
	        	-> (":\"'"&)+: ":\":"
	        		-> ":\"'"&: ":"
	        		-> ":\"'"&: "\""
	        		-> ":\"'"&: ":"
	`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser13(t *testing.T) {
	gram := `
		^S = letter (letter|digit)* digit .
	`
	in := "●aa11"
	out := `
		-> ^S: "aa11"
	        	-> letter: "a"
	        	-> ([letter | digit])*: "a1"
	        		-> [letter | digit]: "a"
	        			-> letter: "a"
	        		-> [letter | digit]: "1"
	        			-> digit: "1"
	        	-> digit: "1"
	`
	testLrparserMode1(t, gram, in, out)
}
func TestLrparser14(t *testing.T) {
	gram := `
		^S = (s2)+ .
		s2 = (letter)+ .
	`
	in := "●aa"
	out := `
		-> ^S: "aa"
	        	-> (s2)+: "aa"
	        		-> s2: "aa"
	        			-> (letter)+: "aa"
	        				-> letter: "a"
	        				-> letter: "a"
	`
	// shift/reduce conflict: either make
	// - several s2, each with one letter
	// - or one s2 with many letters
	//testLrparserMode1(t, gram, in, out) // conflict

	// use shift by default
	testLrparserMode2(t, gram, in, out, false, false, true)
}
func TestLrparser15(t *testing.T) {
	gram := `
		^S = s2~ s3 .
		s2 = "abc" .
		s3 = "de" .
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
		^S = s2~ (s2&)+ .
		s2 = "abc" .
	`
	in := "ab●caa"
	out := `
		-> ^S: "caa"
	        	-> "abc"~: "c"
	        	-> ("abc"&)+: "aa"
	        		-> "abc"&: "a"
	        		-> "abc"&: "a"
	`
	testLrparserMode2(t, gram, in, out, false, false, false)
}

//----------

func TestLrparserStop1(t *testing.T) {
	gram := `
		^id =  digit (letter)* .
	`
	in := "<<<●1ab>>>"
	out := `
		-> ^id: "1ab"
	        	-> digit: "1"
	        	-> (letter)*: "ab"
	        		-> letter: "a"
	        		-> letter: "b"
        `
	testLrparserMode2(t, gram, in, out, false, true, false)
}
func TestLrparserStop2(t *testing.T) {
	gram := `
		^S = letter (linecol)? .
		linecol = e (e)? .
		e = ":" (digit)+ .	
	`
	in := "●a:1:++"
	out := `
		-> ^S: "a:1"
	        	-> letter: "a"
	        	-> (linecol)?: ":1"
	        		-> linecol: ":1"
	        			-> e: ":1"
	        				-> ":": ":"
	        				-> (digit)+: "1"
	        					-> digit: "1"
	        			-> (e)?: ""
	`
	testLrparserMode2(t, gram, in, out, false, true, false)
}
func TestLrparserStop3(t *testing.T) {
	gram := `
		^S = (letter ":")+ .	
	`
	in := "●a:a:a:b"
	out := `
		-> ^S: "a:a:a:"
	        	-> ([letter ":"])+: "a:a:a:"
	        		-> [letter ":"]: "a:"
	        			-> letter: "a"
	        			-> ":": ":"
	        		-> [letter ":"]: "a:"
	        			-> letter: "a"
	        			-> ":": ":"
	        		-> [letter ":"]: "a:"
	        			-> letter: "a"
	        			-> ":": ":"
	`
	testLrparserMode2(t, gram, in, out, false, true, false)
}
func TestLrparserStop3b(t *testing.T) {
	gram := `
		^S = (letter ":")+ (digit)? .
	`
	in := "●a:a:b"
	out := `
		-> ^S: "a:a:"
	        	-> ([letter ":"])+: "a:a:"
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
		^S = (letter digit letter)+ .	
	`
	in := "●a1ab2bc3"
	out := `
		-> ^S: "a1ab2b"
	        	-> ([letter digit letter])+: "a1ab2b"
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
		^S = ("a" "b" "c" "d")+ .		
	`
	in := "●abcdab"
	out := `
		-> ^S: "abcd"
	        	-> (["a" "b" "c" "d"])+: "abcd"
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
		^rev =  digit (letter)*	 .	
		#^rev =  (letter)* digit .
	`
	in := "<<<1ab●>>>"
	//in := "<<<●1ab>>>"
	out := `
		-> ^rev: "1ab"
	        	-> digit: "1"
	        	-> (letter)*: "ab"
	        		-> letter: "a"
	        		-> letter: "b"
        `
	testLrparserMode2(t, gram, in, out, true, true, false)
	//testLrparserMode2(t, gram, in, out, false, true, false) // no rev
}
func TestLrparserRev2(t *testing.T) {
	gram := `
		^S = (s2)? .
		s2 = (letter)+ .
	`
	in := "aa●11"
	out := `
		-> ^S: "aa"
	        	-> (s2)?: "aa"
	        		-> s2: "aa"
	        			-> (letter)+: "aa"
	        				-> letter: "a"
	        				-> letter: "a"
	`
	//testLrparserMode2(t, gram, in, out, true, true)
	testLrparserMode2(t, gram, in, out, true, false, false)
}
func TestLrparserRev3(t *testing.T) {
	gram := `
		^S = (letter|esc)+ .
		esc = "\\" anyrune .		
	`
	in := "d ab\\ c●"
	out := `
		-> ^S: "ab\\ c"
	        	-> ([letter | esc])+: "ab\\ c"
	        		-> [letter | esc]: "a"
	        			-> letter: "a"
	        		-> [letter | esc]: "b"
	        			-> letter: "b"
	        		-> [letter | esc]: "\\ "
	        			-> esc: "\\ "
	        				-> "\\": "\\"
	        				-> anyrune: " "
	        		-> [letter | esc]: "c"
	        			-> letter: "c"
	`
	testLrparserMode2(t, gram, in, out, true, true, false)
}
func TestLrparserRev4(t *testing.T) {
	gram := `
		^S = (letter|esc)* .
		esc = "\\" anyrune .	
	`
	in := "aaa ●bbb"
	out := `
		-> ^S: ""
        		-> ([letter | esc])*: ""
	`
	bnd := testLrparserMode2(t, gram, in, out, true, true, false)
	if bnd.Pos() != 4 {
		t.Fatalf("bad pos: %v", bnd.Pos())
	}
}

//----------

func TestLrparserErr1(t *testing.T) {
	gram := `
		^S = (letter)? letter .	
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
		^S = (digit)+ (letter)? digit . // was endless loop "●111+a"
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
		^S = (digit)+ (letter)? ":" .
		//^S = (digit)+ (letter)? . // ok
	`
	in := "●111a"
	out := ``
	_, err := testLrparserMode3(t, gram, in, out, false, true, true)
	t.Log(err)
	if err == nil {
		t.Fatal("expecting error")
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

	opt := &CPOpt{
		StartRule:         "",
		Reverse:           reverse,
		EarlyStop:         earlyStop,
		ShiftOnSRConflict: shiftOnSRConflict,
		HelperFn:          t.Helper,
		LogfFn:            t.Logf,
	}
	cp, err := lrp.ContentParser(opt)
	if err != nil {
		return nil, err
	}

	bnd, err := cp.Parse([]byte(in2), index)
	if err != nil {
		return nil, err
	}
	res := bnd.SprintRuleTree(-1)

	res2 := testutil.TrimLineSpaces(res)
	expect2 := testutil.TrimLineSpaces(out)
	if res2 != expect2 {
		return nil, fmt.Errorf("%v", res)
	}
	return bnd, nil
}
