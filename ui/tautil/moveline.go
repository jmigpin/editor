package tautil

func MoveLineUp(ta Texta) {
	a, b, hasNewline := linesStringIndexes(ta)
	if a == 0 {
		// already at the first line
		return
	}
	s := ta.Str()[a:b]
	ta.EditOpen()
	defer ta.EditCloseAfterSetCursor()
	ta.EditDelete(a, b)
	a2 := LineStartIndex(ta.Str(), a-1) // previous line, -1 is size of '\n'
	if !hasNewline {
		ta.EditDelete(a-1, a) // remove newline to honor the moving line
		s = s + "\n"
	}
	ta.EditInsert(a2, s)

	if ta.SelectionOn() {
		_, b2, ok := PreviousRuneIndex(ta.Str(), a2+len(s))
		if !ok {
			return
		}
		ta.SetSelection(a2, b2)
	} else {
		// position cursor at same position
		ta.SetCursorIndex(ta.CursorIndex() - (a - a2))
	}
}
func MoveLineDown(ta Texta) {
	a, b, _ := linesStringIndexes(ta)
	if b == len(ta.Str()) {
		// already at the last line
		return
	}
	s := ta.Str()[a:b]
	ta.EditOpen()
	defer ta.EditCloseAfterSetCursor()
	ta.EditDelete(a, b)
	a2, hasNewline := LineEndIndexNextIndex(ta.Str(), a)
	if !hasNewline {
		// remove newline from previous
		s = s[:len(s)-1]
		// insert newline on next
		ta.EditInsert(a2, "\n")
		a2++
	}
	ta.EditInsert(a2, s)

	if ta.SelectionOn() {
		b2 := a2 + len(s)
		if hasNewline {
			var ok bool
			_, b2, ok = PreviousRuneIndex(ta.Str(), b2)
			if !ok {
				return
			}
		}
		ta.SetSelection(a2, b2)
	} else {
		// position cursor at same position
		ta.SetCursorIndex(ta.CursorIndex() + (a2 - a))
	}
}
