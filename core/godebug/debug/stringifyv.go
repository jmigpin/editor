package debug

import (
	"fmt"
	"strconv"
)

func stringifyV(v V) string {
	const max = 150
	str := ""
	switch t := v.(type) {
	case nil:
		return "nil"
	case rune:
		str = fmt.Sprintf("(%q, %d)", t, t)
	case string, error:
		//str = ReducedSprintf(max, "%q", t)
		str = ReducedSprintf(max, "%s", t)
	case fmt.Stringer:
		//str = ReducedSprintf(max, "%q", t.String())
		str = ReducedSprintf(max, "%s", t)
	case float32:
		str = strconv.FormatFloat(float64(t), 'f', -1, 32)
	case float64:
		str = strconv.FormatFloat(t, 'f', -1, 64)
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
		//const q = '"'
		//if rune(s[0]) != q { // keep existing quote if present
		//	s = string(q) + s
		//}
		//s += "..." + string(q)
		s += "..."
	}
	return s
}
