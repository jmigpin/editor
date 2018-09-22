package textutil

import (
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func EndOfLine(te *widget.TextEdit, sel bool) error {
	tc := te.TextCursor

	le, newline, err := iout.LineEndIndex(tc.RW(), tc.Index())
	if err != nil {
		return err
	}
	if newline {
		le--
	}

	tc.SetSelectionUpdate(sel, le)

	return nil
}
