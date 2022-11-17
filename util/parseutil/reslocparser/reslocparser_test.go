package reslocparser

import (
	"testing"

	"github.com/jmigpin/editor/util/testutil"
)

//godebug:annotatepackage:github.com/jmigpin/editor/util/parseutil/lrparser

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

//----------

func TestResLocParserWin1(t *testing.T) {
	in := "++c:\\a\\b.txt:3●"
	out := "c:\\a\\b.txt:3"
	testMode2(t, in, out, '^', '\\', true)
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

	//in = string(bytes.TrimRight(in, "\n"))
	//out = string(bytes.TrimRight(out, "\n"))

	in2, index, err := testutil.SourceCursor("●", string(in), 0)
	if err != nil {
		t.Fatal(err)
	}

	p, err := NewResLocParser()
	if err != nil {
		t.Fatal(err)
	}

	// setup options
	if esc != 0 {
		p.Escape = esc
	}
	if psep != 0 {
		p.PathSeparator = psep
	}
	p.ParseVolume = parseVolume

	if err := p.Init(true); err != nil {
		t.Fatal(err)
	}

	rl, err := p.Parse([]byte(in2), index)
	if err != nil {
		t.Fatal(err)
	}
	res := rl.Stringify1()

	res2 := testutil.TrimLineSpaces(res)
	expect2 := testutil.TrimLineSpaces(out)
	if res2 != expect2 {
		t.Fatalf("res=%v\n%v\n", res, rl.Bnd.SprintRuleTree(5))
	}
}
