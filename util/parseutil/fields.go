package parseutil

import "github.com/jmigpin/editor/util/parseutil/pscan"

func ParseFields(s string, fieldSep rune) ([]string, error) {
	sc := pscan.NewScanner()
	sc.SetSrc([]byte(s))
	esc := '\\'
	fields := []string{}
	if p2, err := sc.M.AndR(0,
		sc.W.LoopSep(
			sc.W.OnValue(
				sc.W.StringValue(sc.W.Loop(sc.W.Or(
					sc.W.EscapeAny(esc),
					sc.W.QuotedString2(esc, 3000, 3000),
					sc.W.RuneNoneOf([]rune{fieldSep}),
				))),
				func(v any) {
					s := v.(string)
					if u, err := UnquoteString(s, esc); err == nil {
						s = u
					}
					s = RemoveEscapes(s, esc)
					fields = append(fields, s)
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
