package tautil

import "strings"

func Replace(ta Texta, old, new string) {
	ta.EditOpen()
	defer ta.EditCloseAfterSetCursor()
	if ta.SelectionOn() {
		a, b := SelectionStringIndexes(ta)
		s := strings.Replace(ta.Str()[a:b], old, new, -1)
		ta.EditDelete(a, b)
		ta.EditInsert(a, s)
		ta.SetCursorIndex(a + len(s))
	} else {
		s := strings.Replace(ta.Str(), old, new, -1)
		ta.EditDelete(0, len(ta.Str()))
		ta.EditInsert(0, s)
	}
}
