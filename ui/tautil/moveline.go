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

	text := ta.Text()

	s := text[a:b]                  // line
	text = text[:a] + text[b:]      // remove line
	a2 := lineStartIndex(text, a-1) // previous line, -1 is size of '\n'
	// add newline if moving the last line
	if b == len(text) {
		s += "\n"
	}
	text = text[:a2] + s + text[a2:] // insert

	ta.SetText(text)

	if ta.SelectionOn() {
		_, b2, ok := PreviousRuneIndex(ta.Text(), a2+len(s))
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
	if b == len(ta.Text()) {
		// already at the last line
		return
	}

	text := ta.Text()

	s := text[a:b]
	text = text[:a] + text[b:] // remove line
	a2 := lineEndIndexNextIndex(text, a)
	text = text[:a2] + s + text[a2:] // insert

	ta.SetText(text)

	if ta.SelectionOn() {
		_, b2, ok := PreviousRuneIndex(ta.Text(), a2+len(s))
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
