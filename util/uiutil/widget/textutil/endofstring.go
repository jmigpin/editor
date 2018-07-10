package textutil

import "github.com/jmigpin/editor/util/uiutil/widget"

func EndOfString(te *widget.TextEdit, sel bool) {
	tc := te.TextCursor
	tc.SetSelectionUpdate(sel, tc.RW().Len())
}
