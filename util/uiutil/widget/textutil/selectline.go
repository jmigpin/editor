package textutil

import (
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func SelectLine(te *widget.TextEdit) error {
	tc := te.TextCursor

	tc.SetSelectionOff()
	a, b, _, err := tc.LinesIndexes()
	if err != nil {
		return err
	}
	tc.SetSelection(a, b)

	// set primary copy
	if tc.SelectionOn() {
		s, err := tc.Selection()
		if err == nil {
			te.SetCPCopy(event.CPIPrimary, string(s))
		}
	}

	return nil
}
