package tautil

import (
	"strings"
)

func Find(ta Texta, str string) bool {
	if str == "" {
		return false
	}
	index, ok := findNextString(ta.Str(), str, ta.CursorIndex())
	if ok {
		i := index + len(str)
		ta.SetSelection(index, i)
		ta.MakeIndexVisibleAtCenter(i)
		return true
	}
	return false
}
func findNextString(text, str string, index int) (int, bool) {
	// ignore case
	str = strings.ToLower(str)
	text = strings.ToLower(text)
	// search from current index
	i := strings.Index(text[index:], str)
	if i >= 0 {
		return index + i, true
	}
	// search from the start
	if index > 0 { // otherwise it would repeat the search above

		lim := index + len(str)
		if lim > len(text) {
			lim = len(text)
		}

		i = strings.Index(text[:lim], str)
		if i >= 0 {
			return i, true
		}
	}
	return 0, false
}
