package tautil

import "unicode"

func MoveCursorJumpRight(ta Texta, sel bool) {
	activateSelection(ta, sel)
	i := moveCursorJumpRightIndex(ta)
	ta.SetCursorIndex(i)
	deactivateSelectionCheck(ta)
}
func moveCursorJumpRightIndex(ta Texta) int {
	i0 := ta.CursorIndex()
	found := false
	foundStop := false
	for ri, ru := range ta.Text()[i0:] {
		if isWordRune(ru) {
			found = true
			if foundStop {
				return i0 + ri
			}
		} else {
			foundStop = true
			if found {
				return i0 + ri
			}
		}
	}
	return len(ta.Text())
}
func MoveCursorJumpLeft(ta Texta, sel bool) {
	activateSelection(ta, sel)
	i := moveCursorJumpLeftIndex(ta)
	ta.SetCursorIndex(i)
	deactivateSelectionCheck(ta)
}
func moveCursorJumpLeftIndex(ta Texta) int {
	found := false
	foundStop := false
	ri := ta.CursorIndex()
	var prevRi int // previous rune index - on the right
	for {
		ru, ri2, ok := PreviousRuneIndex(ta.Text(), ri)
		if !ok {
			break
		}
		prevRi = ri
		ri = ri2

		if isWordRune(ru) {
			found = true
			if foundStop {
				return prevRi
			}
		} else {
			foundStop = true
			if found {
				return prevRi
			}
		}
	}
	return 0
}
func isWordRune(ru rune) bool {
	return unicode.IsLetter(ru) || ru == '_' || unicode.IsDigit(ru)
}
