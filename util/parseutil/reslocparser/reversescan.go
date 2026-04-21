package reslocparser

import (
	"slices"

	"github.com/jmigpin/editor/util/parseutil/btparser"
)

type ReverseScan struct {
	g           btparser.Rules
	fn          btparser.MFn
	escape      rune
	pathSep     rune
	parseVolume bool
}

func NewReverseScanResLoc(escape, pathSep rune, parseVolume bool) *ReverseScan {
	rs := &ReverseScan{}
	rs.g = btparser.NewRules()
	rs.escape = escape
	rs.pathSep = pathSep
	rs.parseVolume = parseVolume
	rs.init()
	return rs
}

func (rs *ReverseScan) ParseStart(src []byte, index, maxLen int) (int, error) {
	ps := btparser.NewParserStateFromBytes(src)
	fn := rs.g.LimitSourceLines(0, 0, rs.g.ReverseSource(rs.fn))
	if maxLen > 0 {
		fn = rs.g.LimitSourceBytes(maxLen, 10, fn)
	}
	p2, err := rs.g.ParseAt(ps, btparser.Pos(index), fn)
	if err != nil {
		return index, err
	}
	return int(p2), nil
}

//----------

func (rs *ReverseScan) init() {
	g := rs.g

	quotes := g.RuneAnyOfString("\"'`")
	pathSeps := []rune{rs.pathSep, '/'}

	except := append([]rune{rs.escape, '/', '\\'}, pathSeps...)
	pathItemSyms := buildPathItemSyms(except...)

	revEscAny := g.ReverseAnd(g.Rune(rs.escape), g.AnyRune())
	revPathItem0 := g.Or(
		g.Rune(rs.escape), // TODO: review?
		revEscAny,
		g.Digit(),
		g.Letter(),
		g.RuneAnyOf(pathItemSyms...),
	)
	revPathItem := func(allowSpace bool) btparser.MFn {
		return g.Or(
			g.And(g.IsTrue(allowSpace), g.Rune(' ')),
			revPathItem0,
		)
	}
	revPath := func(allowSpace bool) btparser.MFn {
		return g.Loop1(g.Or(
			revPathItem(allowSpace),
			g.RuneAnyOf(pathSeps...),
		))
	}

	revVolume := g.And(
		g.IsTrue(rs.parseVolume),
		g.ReverseAnd(g.Letter(), g.Rune(':')),
	)
	revFileScheme := g.ReverseAnd(
		g.SeqOrMid(revStr(fileSchemeTag)),
		g.Optional(g.RuneAnyOf(pathSeps...)),
	)

	revFullPath := func(allowSpace bool) btparser.MFn {
		return g.And(
			g.Optional(revPath(allowSpace)),
			g.Optional(g.And(
				g.Not(revFileScheme),
				revVolume,
			)),
			g.Optional(revFileScheme),
		)
	}

	// Reverse best-effort scanner used only to find a plausible forward parse start near the cursor. Examples of original inputs this tries to cover:
	// "file:///a/b.txt"
	// "file:///a/b.txt:12"
	// "/a/b.txt:12"
	// "/a/b.txt:12:3"
	// "/a/b.txt:o=123"
	// "\"/a/b.txt\", line 23"
	// "/a/b.txt: line 23"
	// "'/a/b c.txt'"
	// The reverse source starts near the cursor, so some tokens can be entered in the middle.That is why multi-rune constants use SeqOrMid and why reverse name parsing tolerates partially aligned escaped sequences and quoted paths.

	//rs.fn = g.DebugAnd(true, "AA",
	rs.fn = g.And(
		// possible line numbers
		g.Optional(g.Loop1(g.Or(g.Digit(), g.Rune(',')))), // <int>,<int>
		g.Optional(g.SeqOrMid("o=")),                      // c offset: "o=<digits>"

		g.Optional(g.Or(
			g.SeqOrMid(revStr(pythonLineTailTag)),
			g.SeqOrMid(revStr(shellLineTailTag)),
			g.Rune(':'),
		)),

		g.Optional(g.Or(
			g.And(
				g.Optional(quotes),            // opt = can be in the middle
				g.Optional(revFullPath(true)), // opt = at last quote
				quotes,
				// verify in the other direction
				g.Peek(g.LimitSourceBytes(800, 800, g.ReverseSource(g.And(
					quotes, revFullPath(true), quotes,
				)))),
			),
			revFullPath(false),
		)),
	)

}

//----------
//----------
//----------

func revStr(s string) string {
	rs := []rune(s)
	slices.Reverse(rs)
	return string(rs)
}
