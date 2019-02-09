package textutil

import (
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func EndOfLine(te *widget.TextEdit, sel bool) error {
	tc := te.TextCursor

	le, newline, err := iorw.LineEndIndex(tc.RW(), tc.Index())
	if err != nil {
		return err
	}
	if newline {
		le--
	}

	tc.SetSelectionUpdate(sel, le)

	return nil
}
