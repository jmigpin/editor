package tautil

import (
	"strings"
	"unicode"
)

func AutoIndent(ta Texta) {
	ci := ta.CursorIndex()
	k := lineStartIndex(ta.Str(), ci)
	if k == ci {
		InsertRune(ta, '\n')
		return
	}

	nonSpace := func(ru rune) bool {
		return !unicode.IsSpace(ru)
	}

	j := strings.IndexFunc(ta.Str()[k:ci], nonSpace)
	if j == 0 {
		InsertRune(ta, '\n')
		return
	} else if j < 0 {
		j = ci - k
	}
	s := "\n" + ta.Str()[k:k+j]
	ta.EditOpen()
	ta.EditInsert(ci, s)
	ta.EditClose()
	ta.SetCursorIndex(ci + len(s))
	ta.MakeIndexVisible(ta.CursorIndex())
}
