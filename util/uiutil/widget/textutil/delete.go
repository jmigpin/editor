package textutil

import (
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func Delete(te *widget.TextEdit) error {
	tc := te.TextCursor
	tc.BeginEdit()
	defer tc.EndEdit()

	var a, b int
	if tc.SelectionOn() {
		a, b = tc.SelectionIndexes()
		tc.SetSelectionOff()
	} else {
		a = tc.Index()
		_, size, err := tc.RW().ReadRuneAt(a)
		if err != nil {
			return err
		}
		b = a + size
	}
	if err := tc.RW().Delete(a, b-a); err != nil {
		return err
	}
	tc.SetIndex(a)
	return nil
}
