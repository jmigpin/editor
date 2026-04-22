package parseutil

import "github.com/jmigpin/editor/util/parseutil/btparser"

func ParseFields(s string, fieldSep rune) ([]string, error) {
	g := btparser.NewRules()
	esc := '\\'
	fields := []string{}

	assignField := func(fn btparser.MFn) btparser.MFn {
		return btparser.AppendLocal(&fields, func(ps *btparser.ParserState, pos btparser.Pos) (string, btparser.MPos, error) {
			s, mp, err := g.VString(fn)(ps, pos)
			if err != nil {
				return "", mp, err
			}
			if u, err := UnquoteString(s, esc); err == nil {
				s = u
			}
			s = RemoveEscapes(s, esc)
			return s, mp, nil
		})
	}

	//----------

	field := g.Loop1(g.Or(
		g.Escape(esc),
		g.QuotedString2(esc, 3000, 3000),
		g.RuneFn(func(ru rune) bool { return ru != fieldSep }),
	))
	fn := g.And(
		g.LoopSep(false,
			assignField(field),
			g.Rune(fieldSep),
		),
		g.Eof(),
	)

	ps := btparser.NewParserStateFromString(s)
	if _, err := g.Parse(ps, fn); err != nil {
		return nil, err
	}
	return fields, nil
}
