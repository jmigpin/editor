package parseutil

import "fmt"

func ParseFields(s string, fieldSep rune) ([]string, error) {
	sc := NewScanner()
	sc.SetSrc([]byte(s))
	esc := '\\'
	fields := []string{}
	for i := 0; ; i++ {
		if sc.M.Eof() {
			break
		}

		// field separator
		if i > 0 {
			if err := sc.M.Rune(fieldSep); err != nil {
				return nil, fmt.Errorf("field separator: %w", err)
			}
		}

		// field (can be empty)
		pos0 := sc.Pos
		for {
			if sc.M.QuotedString2(esc, 3000, 3000) == nil {
				continue
			}
			if sc.M.RuneAnyNot([]rune{fieldSep}) == nil {
				continue
			}
			break
		}
		val := string(sc.Src[pos0:sc.Pos])
		if u, err := UnquoteString(val, esc); err == nil {
			val = u
		}

		// add field
		fields = append(fields, val)
	}
	return fields, nil
}
