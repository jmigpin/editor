package tautil

import "strings"

func EndOfLine(ta Texta, sel bool) {
	ci := ta.CursorIndex()
	i := strings.Index(ta.Str()[ci:], "\n")
	if i < 0 {
		i = len(ta.Str())
	} else {
		i += ci
	}
	updateSelection(ta, sel, i)
}
