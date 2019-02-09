package textutil

import (
	"unicode"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func StartOfLine(te *widget.TextEdit, sel bool) error {
	tc := te.TextCursor

	ci := tc.Index()
	i, err := iorw.LineStartIndex(tc.RW(), ci)
	if err != nil {
		return err
	}

	// stop at first non blank rune from the left
	n := ci - i
	for j := 0; j < n; j++ {
		ru, _, err := tc.RW().ReadRuneAt(i + j)
		if err != nil {
			return err
		}
		if !unicode.IsSpace(ru) {
			i += j
			break
		}
	}

	tc.SetSelectionUpdate(sel, i)
	return nil
}
