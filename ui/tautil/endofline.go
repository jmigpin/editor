package tautil

import "strings"

func EndOfLine(ta Texta, sel bool) {
	activateSelection(ta, sel)
	i := strings.Index(ta.Text()[ta.CursorIndex():], "\n")
	if i < 0 {
		i = len(ta.Text())
	} else {
		i += ta.CursorIndex()
	}
	ta.SetCursorIndex(i)
	deactivateSelectionCheck(ta)
}
