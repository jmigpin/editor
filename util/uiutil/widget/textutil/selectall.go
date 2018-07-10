package textutil

import "github.com/jmigpin/editor/util/uiutil/widget"

func SelectAll(te *widget.TextEdit) {
	tc := te.TextCursor
	tc.SetSelection(0, tc.RW().Len())
}
