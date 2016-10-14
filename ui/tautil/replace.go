package tautil

import "strings"

func Replace(ta Texta, old, new string) {
	if ta.SelectionOn() {
		a, b, ok := selectionStringIndexes(ta)
		if !ok {
			return
		}
		s := strings.Replace(ta.Str()[a:b], old, new, -1)
		ta.EditRemove(a, b)
		ta.EditInsert(a, s)
		ta.EditCommit()
		ta.SetCursorIndex(a + len(s))
	} else {
		s := strings.Replace(ta.Str(), old, new, -1)
		ta.EditRemove(0, len(ta.Str()))
		ta.EditInsert(0, s)
		ta.EditCommit()
	}
}
