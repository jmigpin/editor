package tautil

import "unicode"

func StartOfLine(ta Texta, sel bool) {
	activateSelection(ta, sel)
	defer deactivateSelectionCheck(ta)

	i := lineStartIndex(ta.Str(), ta.CursorIndex())

	// stop at first non blank rune from the left
	t := ta.Str()[i:ta.CursorIndex()]
	for j, ru := range t {
		if !unicode.IsSpace(ru) {
			i += j
			break
		}
	}

	ta.SetCursorIndex(i)
}
