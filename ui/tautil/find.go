package tautil

import (
	"strings"
)

func Find(ta Texta, str string) {
	if str == "" {
		return
	}
	index, ok := findNextString(ta.Text(), str, ta.CursorIndex())
	if ok {
		ta.SetSelectionOn(true)
		ta.SetSelectionIndex(index)
		i := index + len(str)
		ta.SetCursorIndex(i)
		ta.MakeIndexVisible(i)
	}
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
		i = strings.Index(text[:index], str)
		if i >= 0 {
			return i, true
		}
	}
	return 0, false
}
