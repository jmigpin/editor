package debug

import (
	"fmt"
	"strconv"
)

func stringifyV(v V) string {
	// Note: rune is an alias for int32, can't "case rune:"
	const max = 150
	str := ""
	switch t := v.(type) {
	case nil:
		return "nil"
	case string:
		str = ReducedSprintf(max, "%q", t)
	case []string:
		str = quotedStrings(max, t)
	case fmt.Stringer:
		str = ReducedSprintf(max, "%q", t)
	case error:
		str = ReducedSprintf(max, "%q", t)
	case float32:
		str = strconv.FormatFloat(float64(t), 'f', -1, 32)
	case float64:
		str = strconv.FormatFloat(t, 'f', -1, 64)
	default:
		str = ReducedSprintf(max, "%v", v) // ex: bool
	}
	return str
}

func ReducedSprintf(max int, format string, a ...interface{}) string {
	w := NewLimitedWriter(max)
	_, err := fmt.Fprintf(w, format, a...)
	s := string(w.Bytes())
	if err == LimitReachedErr {
		s += "..."
		// close quote if present
		const q = '"'
		if rune(s[0]) == q {
			s += string(q)
		}
	}
	return s
}

func quotedStrings(max int, a []string) string {
	w := NewLimitedWriter(max)
	sp := ""
	limited := 0
	for i, s := range a {
		if i > 0 {
			sp = " "
		}
		n, err := fmt.Fprintf(w, "%s%q", sp, s)
		if err != nil {
			if err == LimitReachedErr {
				limited = n
			}
			break
		}
	}
	s := string(w.Bytes())
	if limited > 0 {
		s += "..."
		if limited >= 2 { // 1=space, 2=quote
			s += `"` // close quote
		}
	}
	return "[" + s + "]"
}
