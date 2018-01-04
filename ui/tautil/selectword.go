package tautil

import (
	"strings"

	"github.com/jmigpin/editor/util/uiutil/event"
)

func SelectWord(ta Texta) {
	index := ta.CursorIndex()
	a := wordLeftIndex(ta.Str(), index)
	b := wordRightIndex(ta.Str(), index)
	ta.SetSelection(a, b)

	// set primary copy
	if ta.SelectionOn() {
		a, b := SelectionStringIndexes(ta)
		s := ta.Str()[a:b]
		err := ta.SetCPCopy(event.PrimaryCPI, s)
		if err != nil {
			ta.Error(err)
		}
	}
}

func wordLeftIndex(str string, index int) int {
	typ := 0

	ru, _, ok := NextRuneIndex(str, index)
	if ok {
		typ = wordType(ru)
	}

	fn := func(ru rune) bool {
		typ2 := wordType(ru)
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
	fn := func(ru rune) bool {
		typ2 := wordType(ru)
		if typ == 0 {
			typ = typ2
			return false
		}
		return typ2 != typ
	}
	i := strings.IndexFunc(str[index:], fn)
	if i < 0 {
		i = len(str[index:])
	}
	return index + i
}
func wordType(ru rune) int {
	if isWordRune(ru) {
		return 1
	}
	return 2 + int(ru)
}
