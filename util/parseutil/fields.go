package parseutil

import "github.com/jmigpin/editor/util/parseutil/pscan"

func ParseFields(s string, fieldSep rune) ([]string, error) {
	sc := pscan.NewScanner()
	sc.SetSrc([]byte(s))
	esc := '\\'
	fields := []string{}
	if p2, err := sc.M.And(0,
		sc.W.LoopSep(
			false,
			pscan.WOnValueM(
				sc.W.StrValue(sc.W.LoopOneOrMore(sc.W.Or(
					sc.W.EscapeAny(esc),
					sc.W.QuotedString2(esc, 3000, 3000),
					sc.W.RuneNoneOf([]rune{fieldSep}),
				))),
				func(s string) error {
					if u, err := UnquoteString(s, esc); err == nil {
						s = u
					}
					s = RemoveEscapes(s, esc)
					fields = append(fields, s)
					return nil
				},
			),
			// separator
			sc.W.Rune(fieldSep),
		),
		sc.M.Eof,
	); err != nil {
		return nil, sc.SrcError(p2, err)
	} else {
		return fields, nil
	}
}
