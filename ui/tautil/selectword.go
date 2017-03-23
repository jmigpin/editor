package tautil

import "strings"

func SelectWord(ta Texta) {
	index := ta.CursorIndex()
	a := wordLeftIndex(ta.Str(), index)
	b := wordRightIndex(ta.Str(), index)
	if a != b {
		ta.SetSelectionOn(true)
		ta.SetSelectionIndex(a)
	}
	ta.SetCursorIndex(b)
}
func wordLeftIndex(str string, index int) int {
	typ := 0

	getType := func(ru rune) int {
		if isWordRune(ru) {
			return 1
		}
		return 2
	}

	ru, _, ok := NextRuneIndex(str, index)
	if ok {
		typ = getType(ru)
	}

	fn := func(ru rune) bool {
		typ2 := getType(ru)
		if typ == 0 {
			typ = typ2
			return false
		}
		return typ2 != typ
	}
	i := strings.LastIndexFunc(str[:index], fn)
	if i < 0 {
		i = 0
	} else {
		i++
	}
	return i
}
func wordRightIndex(str string, index int) int {
	typ := 0

	getType := func(ru rune) int {
		if isWordRune(ru) {
			return 1
		}
		return 2
	}

	fn := func(ru rune) bool {
		typ2 := getType(ru)
		if typ == 0 {
			typ = typ2
			return false
		}
		return typ2 != typ
	}
	i := strings.IndexFunc(str[index:], fn)
	if i < 0 {
		i = index
	}
	return index + i
}
