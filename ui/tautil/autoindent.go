package tautil

import "strings"

func AutoIndent(ta Texta) {
	// string to insert
	ci := ta.CursorIndex()
	k := LineStartIndex(ta.Str(), ci)
	j := strings.IndexFunc(ta.Str()[k:ci], isNotSpace)
	if j < 0 {
		// full line of spaces, indent to cursor position
		j = ci - k
	}
	str := "\n" + ta.Str()[k:k+j]

	ta.EditOpen()
	if ta.SelectionOn() {
		// remove selection
		a, b := SelectionStringIndexes(ta)
		ta.EditDelete(a, b)
		ta.SetSelectionOff()
		ta.SetCursorIndex(a)
	}
	// insert
	ta.EditInsert(ta.CursorIndex(), str)
	ta.SetCursorIndex(ta.CursorIndex() + len(str))
	ta.EditCloseAfterSetCursor()
}
