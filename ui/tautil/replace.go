package tautil

import "strings"

func Replace(ta Texta, old, new string) {
	if ta.SelectionOn() {
		a, b := SelectionStringIndexes(ta)
		s := strings.Replace(ta.Str()[a:b], old, new, -1)
		ta.EditOpen()
		ta.EditDelete(a, b)
		ta.EditInsert(a, s)
		ta.EditClose()
		ta.SetCursorIndex(a + len(s))
	} else {
		s := strings.Replace(ta.Str(), old, new, -1)
		ta.EditOpen()
		ta.EditDelete(0, len(ta.Str()))
		ta.EditInsert(0, s)
		ta.EditClose()
	}
}
