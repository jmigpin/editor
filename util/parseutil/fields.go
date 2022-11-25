package parseutil

import "fmt"

func ParseFields(s string, fieldSep rune) ([]string, error) {
	ps := NewPState([]byte(s))
	esc := '\\'
	fields := []string{}
	for i := 0; ; i++ {
		if ps.MatchEof() == nil {
			break
		}

		// field separator
		if i > 0 {
			if err := ps.MatchRune(fieldSep); err != nil {
				return nil, fmt.Errorf("field separator: %w", err)
			}
		}

		// field (can be empty)
		pos0 := ps.Pos
		for {
			if ps.QuotedString2(esc, 3000, 3000) == nil {
				continue
			}
			if ps.MatchRunesOrNeg([]rune{fieldSep}) == nil {
				continue
			}
			break
		}
		val := string(ps.Src[pos0:ps.Pos])
		if u, err := UnquoteString(val, esc); err == nil {
			val = u
		}

		// add field
		fields = append(fields, val)
	}
	return fields, nil
}
