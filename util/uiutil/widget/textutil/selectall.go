package textutil

import "github.com/jmigpin/editor/util/uiutil/widget"

func SelectAll(te *widget.TextEdit) {
	tc := te.TextCursor
	tc.SetSelection(tc.RW().Min(), tc.RW().Max())
}
