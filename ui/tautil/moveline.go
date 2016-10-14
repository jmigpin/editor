package tautil

func MoveLineUp(ta Texta) {
	a, b, ok := linesStringIndexes(ta)
	if !ok {
		return
	}
	if a == 0 {
		// already at the first line
		return
	}

	s := ta.Str()[a:b] // line
	// add newline if moving the last line
	if b == len(ta.Str()) {
		s += "\n"
	}
	ta.EditRemove(a, b)
	a2 := lineStartIndex(ta.Str(), a-1) // previous line, -1 is size of '\n'
	ta.EditInsert(a2, s)
	ta.EditCommit()

	if ta.SelectionOn() {
		_, b2, ok := PreviousRuneIndex(ta.Str(), a2+len(s))
		if !ok {
			return
		}
		ta.SetSelectionIndex(a2)
		ta.SetCursorIndex(b2)
	} else {
		// position cursor at same position
		ta.SetCursorIndex(ta.CursorIndex() - (a - a2))
	}
}
func MoveLineDown(ta Texta) {
	a, b, ok := linesStringIndexes(ta)
	if !ok {
		return
	}
	if b == len(ta.Str()) {
		// already at the last line
		return
	}

	s := ta.Str()[a:b]
	ta.EditRemove(a, b)
	a2 := lineEndIndexNextIndex(ta.Str(), a)
	ta.EditInsert(a2, s)
	ta.EditCommit()

	if ta.SelectionOn() {
		_, b2, ok := PreviousRuneIndex(ta.Str(), a2+len(s))
		if !ok {
			return
		}
		ta.SetSelectionIndex(a2)
		ta.SetCursorIndex(b2)
	} else {
		// position cursor at same position
		ta.SetCursorIndex(ta.CursorIndex() + (a2 - a))
	}
}
