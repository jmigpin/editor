package reslocparser

import "github.com/jmigpin/editor/util/parseutil/btparser"

type ReverseScan struct {
	g  btparser.Rules
	fn btparser.MFn

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

func (rs *ReverseScan) ParseStart(ps *btparser.ParserState, index, maxLen int) (int, error) {
	p2, err := rs.g.ParseAt(ps, btparser.Pos(index), rs.Rule(maxLen))
	return int(p2), err
}

func (rs *ReverseScan) Rule(maxLen int) btparser.MFn {
	return rs.g.WithBounds(maxLen, 10,
		rs.g.WithLineBounds(0, 0, rs.fn),
	)
}

//----------

func (rs *ReverseScan) init() {
	g := rs.g

	quotes := g.RuneAnyOfString("\"'`")
	pathSeps := []rune{rs.pathSep, '/'}

	except := append([]rune{rs.escape, '/', '\\'}, pathSeps...)
	pathItemSyms := buildPathItemSyms(except...)

	revEscAny := g.And(g.AnyRune(), g.Rune(rs.escape))
	revPathItem0 := g.Or(
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
		g.And(g.Rune(':'), g.Letter()),
	)
	revFileScheme := g.And(
		g.Optional(g.RuneAnyOf(pathSeps...)),
		g.SeqOrMid(btparser.ReverseString(fileSchemeTag)),
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
	rs.fn = rs.g.ReverseSource(g.And(
		// improve pos to avoid being in the middle of an escape
		g.Optional(g.Loop1(g.Rune(rs.escape))),

		// possible line numbers
		g.Optional(g.Loop1(g.Or(g.Digit(), g.Rune(',')))), // <int>,<int>
		g.Optional(g.SeqOrMid("o=")),                      // c offset: "o=<digits>"

		g.Optional(g.Or(
			g.SeqOrMid(btparser.ReverseString(pythonLineTailTag)),
			g.SeqOrMid(btparser.ReverseString(shellLineTailTag)),
			g.Rune(':'),
		)),

		g.Optional(g.Or(
			g.And(
				g.Optional(quotes),            // opt = can be in the middle
				g.Optional(revFullPath(true)), // opt = at last quote
				quotes,
				// verify in the other direction
				g.Peek(g.WithBounds(800, 800,
					g.ReverseSource(g.And(
						quotes, revFullPath(true), quotes,
					)),
				)),
			),
			revFullPath(false),
		)),
	))

}

//----------
//----------
//----------

// coverIndex brute-forces possible parse starts from pos and accepts the first match that reaches index, useful as a simple fallback when reverse scanning cannot reliably find the start.
func coverIndex(index int, fn btparser.MFn) btparser.MFn {
	return func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
		return coverIndexParse(ps, pos, index, fn)
	}
}

func coverIndexParse(ps *btparser.ParserState, pos btparser.Pos, index int, fn btparser.MFn) (btparser.MPos, error) {
	var err0 error
	for i := int(pos); i <= index; i++ {
		rl1 := coverIndexResLocData(ps)
		rl2 := *rl1
		ps.UserData[resLocDataKey] = &rl2

		mp, err := fn(ps, btparser.Pos(i))
		ps.UserData[resLocDataKey] = rl1
		if err != nil {
			if err0 == nil {
				err0 = err
			}
			continue
		}
		if int(mp.End) < index {
			continue
		}

		*rl1 = rl2
		return btparser.MPos{Start: btparser.Pos(i), End: mp.End}, nil
	}
	if err0 != nil {
		return btparser.MPos{}, err0
	}
	return btparser.MPos{}, btparser.NoMatchErr
}

func coverIndexResLocData(ps *btparser.ParserState) *ResLoc {
	rl, ok := ps.UserData[resLocDataKey].(*ResLoc)
	if !ok {
		panic("cover index scan missing ResLoc userdata")
	}
	return rl
}
