package cmdutil

import (
	"strings"
	"unicode"

	"github.com/jmigpin/editor/ui"
)

func OpenFileLineAtCol(ed Editori, filename string, line int, col *ui.Column) {
	row, err := ed.FindRowOrCreateInColFromFilepath(filename, col)
	if err != nil {
		ed.Error(err)
		return
	}
	// don't search/touch the indexes if the line is not set (zero)
	if line == 0 {
		row.Square.WarpPointer()
		return
	}
	// find line
	ta := row.TextArea
	index := 0
	line--
	for i, ru := range ta.Str() {
		if ru == '\n' {
			line--
			if line == 0 {
				index = i + 1
				break
			}
		}
	}
	// extra: go to first non empty char
	if index > 0 {
		isNotSpace := func(ru rune) bool { return !unicode.IsSpace(ru) }
		j := strings.IndexFunc(ta.Str()[index:], isNotSpace)
		if j > 0 {
			index += j
		}
	}

	ta.SetSelectionOn(false)
	ta.SetCursorIndex(index)
	ta.MakeCursorVisibleAndWarpPointerToCursor()
}
