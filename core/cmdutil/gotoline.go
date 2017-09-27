package cmdutil

import (
	"fmt"
	"strconv"

	"github.com/jmigpin/editor/core/toolbardata"
	"github.com/jmigpin/editor/ui"
)

func GotoLine(erow ERower, part *toolbardata.Part) {
	a := part.Args[1:]
	if len(a) != 1 {
		err := fmt.Errorf("gotoline: expecting 1 argument")
		erow.Ed().Error(err)
		return
	}

	line, err := strconv.ParseUint(a[0].Str, 10, 64)
	if err != nil {
		erow.Ed().Error(err)
		return
	}

	GotoLineColumnInTextArea(erow.Row().TextArea, int(line), 0)
}

func GotoLineColumnInTextArea(ta *ui.TextArea, line, column int) {
	line--
	column--

	// rune index of line/column
	index := 0
	l, c := 0, 0
	for ri, ru := range ta.Str() {
		if l == line {
			if c == column {
				index = ri
				break
			}
			c++
		}
		if ru == '\n' {
			l++
			if l == line {
				index = ri + 1 // column 0 (+1 is to pass '\n')
			}
			if l > line {
				break
			}
		}
	}

	// goto index
	ta.SetSelectionOff()
	ta.SetCursorIndex(index)
	ta.MakeIndexVisibleAtCenter(index)
	ta.WarpPointerToIndexIfVisible(index)
}
