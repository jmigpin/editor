package textutil

import (
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func Cut(te *widget.TextEdit) error {
	tc := te.TextCursor

	if !tc.SelectionOn() {
		return nil
	}

	tc.BeginEdit()
	defer tc.EndEdit()

	a, b := tc.SelectionIndexes()
	s, err := tc.RW().ReadNCopyAt(a, b-a)
	if err != nil {
		return err
	}
	te.SetCPCopy(event.CPIClipboard, string(s))

	if err := tc.RW().Delete(a, b-a); err != nil {
		return err
	}
	tc.SetSelectionOff()
	tc.SetIndex(a)
	return nil
}
