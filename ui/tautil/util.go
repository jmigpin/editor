package tautil

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func isNotSpace(ru rune) bool {
	return !unicode.IsSpace(ru)
}
func isWordRune(ru rune) bool {
	return unicode.IsLetter(ru) || ru == '_' || unicode.IsDigit(ru)
}

func updateSelectionState(ta Texta, active bool) {
	if active {
		if !ta.SelectionOn() {
			ta.SetSelectionOn(true)
			ta.SetSelectionIndex(ta.CursorIndex())
		}
	} else {
		ta.SetSelectionOn(false)
	}
}

func NextRuneIndex(str string, index int) (rune, int, bool) {
	ru, size := utf8.DecodeRuneInString(str[index:])
	if ru == utf8.RuneError {
		if size == 0 { // empty string
			return 0, 0, false
		}
		// size==1// invalid encoding, continue with 1
		ru = rune(str[index+size])
	}
	return ru, index + size, true
}
func PreviousRuneIndex(str string, index int) (rune, int, bool) {
	ru, size := utf8.DecodeLastRuneInString(str[:index])
	if ru == utf8.RuneError {
		if size == 0 { // empty string
			return 0, 0, false
		}
		// size==1 // invalid encoding, continue with 1
		ru = rune(str[index-size])
	}
	return ru, index - size, true
}

func SelectionStringIndexes(ta Texta) (int, int) {
	if !ta.SelectionOn() {
		panic("selection should be on")
	}
	a := ta.SelectionIndex()
	b := ta.CursorIndex()
	if a > b {
		a, b = b, a
	}
	return a, b
}

func linesStringIndexes(ta Texta) (int, int, bool) {
	var a, b int
	if ta.SelectionOn() {
		a, b = SelectionStringIndexes(ta)
	} else {
		a = ta.CursorIndex()
		b = a
	}
	a = lineStartIndex(ta.Str(), a)
	b, hasNewline := lineEndIndexNextIndex(ta.Str(), b)
	return a, b, hasNewline
}

func lineStartIndex(str string, index int) int {
	i := strings.LastIndex(str[:index], "\n")
	if i < 0 {
		i = 0
	} else {
		i += 1 // rune length of '\n'
	}
	return i
}
func lineEndIndexNextIndex(str string, index int) (_ int, hasNewline bool) {
	i := strings.Index(str[index:], "\n")
	if i < 0 {
		return len(str), false
	}
	return index + i + 1, true // 1 is "\n" size
}
