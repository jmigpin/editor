package textutil

import "github.com/jmigpin/editor/util/uiutil/widget"

func StartOfString(te *widget.TextEdit, sel bool) {
	te.TextCursor.SetSelectionUpdate(sel, 0)
}
