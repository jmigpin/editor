package textutil

import "unicode"

//----------

// Used at: selectword, movecursorjump{left,right}
func isWordRune(ru rune) bool {
	return unicode.IsLetter(ru) || ru == '_' || unicode.IsDigit(ru)
}
