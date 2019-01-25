package debug

import (
	"fmt"
	"unicode"
)

func stringifyV(v V) string {
	const max = 150
	str := ""
	switch t := v.(type) {
	case nil:
		return "nil"
	case rune:
		if unicode.IsGraphic(t) {
			str = fmt.Sprintf("(%q=%d)", t, t)
		} else {
			str = fmt.Sprintf("%v", t)
		}
	case string:
		str = ReducedSprintf(max, "%q", t)
	case fmt.Stringer, error:
		str = ReducedSprintf(max, "%q", t) // used to be ≈(%q)

	case float32, float64:
		u := fmt.Sprintf("%f", t)

		// reduce trailing zeros
		j := 0
		for i := len(u) - 1; i >= 0; i-- {
			if u[i] == '0' {
				j++
				continue
			}
			break
		}

		str = u[:len(u)-j]

	default:
		str = ReducedSprintf(max, "%v", v)
	}

	return str
}

func ReducedSprintf(max int, format string, a ...interface{}) string {
	w := NewLimitedWriter(max)
	_, err := fmt.Fprintf(w, format, a...)
	s := string(w.Bytes())
	if err != nil {
		if s[0] != '"' { // keep existing quote if present
			s = "\"" + s
		}
		s += "...\"" // "◦◦◦"
	}
	return s
}
