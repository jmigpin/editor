package reslocparser

import (
	"testing"

	"github.com/jmigpin/editor/util/testutil"
)

func TestResLocParser1(t *testing.T) {
	in := "AAA /a/b/c●.txt BBB"
	out := "/a/b/c.txt"
	testMode1(t, in, out)
}
func TestResLocParser2(t *testing.T) {
	in := "AAA /a/b/●c%20c.txt AAA"
	out := "/a/b/c c.txt"
	testMode1(t, in, out)
}
func TestResLocParser3(t *testing.T) {
	in := "AAA /a/b/●c\\ c.txt AAA"
	out := "/a/b/c\\ c.txt"
	testMode1(t, in, out)
}
func TestResLocParser4(t *testing.T) {
	in := "AAA /a/b/●c\\\\ c.txt AAA"
	out := "/a/b/c\\\\"
	testMode1(t, in, out)
}
func TestResLocParser5(t *testing.T) {
	in := "/a/b/c\\● c.txt"
	out := "/a/b/c\\ c.txt"
	testMode1(t, in, out)
}
func TestResLocParser5b(t *testing.T) {
	in := "/a/b/c\\\\\\ \\ ●c.txt"
	out := "/a/b/c\\\\\\ \\ c.txt"
	testMode1(t, in, out)
}
func TestResLocParser5c(t *testing.T) {
	in := "AAA /a/b/c\\ ●c.txt AAA"
	out := "/a/b/c\\ c.txt"
	testMode1(t, in, out)
}
func TestResLocParser5d(t *testing.T) {
	in := "AAA /a/b/c\\ c●.txt AAA"
	out := "/a/b/c\\ c.txt"
	testMode1(t, in, out)
}
func TestResLocParser5e(t *testing.T) {
	in := "●/a/b/c\\ c.txt AAA"
	out := "/a/b/c\\ c.txt"
	testMode1(t, in, out)
}
func TestResLocParser5f(t *testing.T) {
	in := " a\\● b\\ c.txt"
	out := "a\\ b\\ c.txt"
	testMode1(t, in, out)
}
func TestResLocParser6(t *testing.T) {
	in := "AAA /a/b/c.●txt\\:a:1:2# AAA"
	out := "/a/b/c.txt\\:a:1:2"
	testMode1(t, in, out)
}
func TestResLocParser7(t *testing.T) {
	in := "AAA /a/b/c.●txt\\:a:1:#AAA"
	out := "/a/b/c.txt\\:a:1"
	testMode1(t, in, out)
}
func TestResLocParser8(t *testing.T) {
	in := "●/a/b/c:1:2"
	out := "/a/b/c:1:2"
	testMode1(t, in, out)
}
func TestResLocParser9(t *testing.T) {
	in := "●/a/b\\ b/c"
	out := "/a/b\\ b/c"
	testMode1(t, in, out)
}
func TestResLocParser10(t *testing.T) {
	in := "●/a/b\\"
	out := "/a/b"
	testMode1(t, in, out)
}
func TestResLocParser11(t *testing.T) {
	in := ": /a/b/c●"
	out := "/a/b/c"
	testMode1(t, in, out)
}
func TestResLocParser12(t *testing.T) {
	in := "//a/b/////c●"
	out := "/a/b/c"
	testMode1(t, in, out)
}
func TestResLocParser13(t *testing.T) {
	in := "(/a/b●/c.txt)"
	out := "/a/b/c.txt"
	testMode1(t, in, out)
}
func TestResLocParser14(t *testing.T) {
	in := "[/a/b●/c.txt]"
	out := "/a/b/c.txt"
	testMode1(t, in, out)
}
func TestResLocParser15(t *testing.T) {
	in := "</a/b●/c.txt>"
	out := "/a/b/c.txt"
	testMode1(t, in, out)
}
func TestResLocParser17(t *testing.T) {
	in := "./a●/b/c.txt :20"
	out := "./a/b/c.txt"
	testMode1(t, in, out)
}
func TestResLocParser18(t *testing.T) {
	in := "aa ●file:///a/b/c.txt bb"
	out := "/a/b/c.txt"
	testMode1(t, in, out)
}
func TestResLocParser18b(t *testing.T) {
	in := "aa file://●/a/b/c.txt bb"
	out := "/a/b/c.txt"
	testMode1(t, in, out)
}
func TestResLocParser18c(t *testing.T) {
	in := "aa file:///a/b/●c.txt bb"
	out := "/a/b/c.txt"
	testMode1(t, in, out)
}
func TestResLocParser19(t *testing.T) {
	in := "aa &{file:///a/●b/c.txt}"
	out := "/a/b/c.txt"
	testMode1(t, in, out)
}
func TestResLocParser20(t *testing.T) {
	in := "aa &{file:///a/●b/c%2b%2b.txt}"
	out := "/a/b/c++.txt"
	testMode1(t, in, out)
}
func TestResLocParser21(t *testing.T) {
	in := "-arg=/a/●b/c.txt"
	out := "/a/b/c.txt"
	testMode1(t, in, out)
}
func TestResLocParser22(t *testing.T) {
	in := "/a/b/●!u!w.txt"
	out := "/a/b/!u!w.txt"
	testMode1(t, in, out)
}
func TestResLocParser23(t *testing.T) {
	in := "\"a/b/c.txt\", line 10●"
	out := "a/b/c.txt:10"
	testMode1(t, in, out)
}
func TestResLocParser24(t *testing.T) {
	in := "bbb \"aa.py\", line● 10, in <bb>"
	out := "aa.py:10"
	testMode1(t, in, out)
}
func TestResLocParser25(t *testing.T) {
	in := "bbb \"aa.py\", line 10●, in <bb>"
	out := "aa.py:10"
	testMode1(t, in, out)
}
func TestResLocParser26(t *testing.T) {
	in := "bbb \"a●a.py\", line 10, in <bb>"
	out := "aa.py:10"
	testMode1(t, in, out)
}
func TestResLocParser27(t *testing.T) {
	in := "bbb \"a●a.py\" bbb"
	out := "aa.py"
	testMode1(t, in, out)
}
func TestResLocParser28(t *testing.T) {
	in := "/a/b.txt:●3"
	out := "/a/b.txt:3"
	testMode1(t, in, out)
}
func TestResLocParser29(t *testing.T) {
	in := "file:/●//a/b.txt"
	out := "/a/b.txt"
	testMode1(t, in, out)
}
func TestResLocParser30(t *testing.T) {
	in := "●file:///a/b.txt:1:1"
	out := "/a/b.txt:1:1"
	testMode1(t, in, out)
}
func TestResLocParser31(t *testing.T) {
	in := "/a/b.t●xt: line 2: etc"
	out := "/a/b.txt:2"
	testMode1(t, in, out)
}
func TestResLocParser32(t *testing.T) {
	in := "/a/b.txt: line ●2: etc"
	out := "/a/b.txt:2"
	testMode1(t, in, out)
}
func TestResLocParser33(t *testing.T) {
	in := "\"/a/b.●txt\""
	out := "/a/b.txt"
	testMode1(t, in, out)
}
func TestResLocParser34(t *testing.T) {
	in := "/a/b.●txt:o=5"
	out := "/a/b.txt:o=5"
	testMode2b(t, in, out, 0, 0, false)
}

//----------

func TestResLocParserWin1(t *testing.T) {
	in := "++c:\\a\\b.t^ xt:3●"
	out := "c:\\a\\b.t^ xt:3"
	testMode2(t, in, out, '^', '\\', true)
}
func TestResLocParserWin2(t *testing.T) {
	in := "file:///c:/a/b.txt:3●"
	out := "c:\\a\\b.txt:3"
	testMode2(t, in, out, '^', '\\', true)
}
func TestResLocParserWin3(t *testing.T) {
	in := "..\\\nabc\\●"
	out := "abc\\"
	testMode2(t, in, out, '^', '\\', true)
}

//----------
//----------
//----------

func BenchmarkResLoc1(b *testing.B) {
	t := b
	in := "/a/b/c.●txt:1:2"
	in2, index, err := testutil.SourceCursor("●", string(in), 0)
	if err != nil {
		t.Fatal(err)
	}

	p := NewResLocParser()
	p.Init()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl, err := p.Parse([]byte(in2), index)
		if err != nil {
			t.Fatal(err)
		}
		_ = rl
	}
}

//----------
//----------
//----------

func testMode1(t *testing.T, in, out string) {
	t.Helper()
	testMode2(t, in, out, 0, 0, false)
}

func testMode2(t *testing.T, in, out string, esc, psep rune, parseVolume bool) {
	t.Helper()
	rl := testMode3(t, in, out, esc, psep, parseVolume)
	res := rl.Stringify1()
	res2 := testutil.TrimLineSpaces(res)
	expect2 := testutil.TrimLineSpaces(out)
	if res2 != expect2 {
		//t.Fatalf("res=%v\n%v\n", res, rl.Bnd.SprintRuleTree(5))
		t.Fatalf("res=%v", res)
	}
}
func testMode2b(t *testing.T, in, out string, esc, psep rune, parseVolume bool) {
	t.Helper()
	rl := testMode3(t, in, out, esc, psep, parseVolume)
	res := rl.ToOffsetString()
	res2 := testutil.TrimLineSpaces(res)
	expect2 := testutil.TrimLineSpaces(out)
	if res2 != expect2 {
		t.Fatalf("res=%v", res)
	}
}

func testMode3(t *testing.T, in, out string, esc, psep rune, parseVolume bool) *ResLoc {
	t.Helper()

	in2, index, err := testutil.SourceCursor("●", string(in), 0)
	if err != nil {
		t.Fatal(err)
	}

	p := NewResLocParser()

	// setup options
	if esc != 0 {
		p.Escape = esc
	}
	if psep != 0 {
		p.PathSeparator = psep
	}
	p.ParseVolume = parseVolume

	// parse
	p.Init()
	rl, err := p.Parse([]byte(in2), index)
	if err != nil {
		t.Fatal(err)
	}
	return rl
}
