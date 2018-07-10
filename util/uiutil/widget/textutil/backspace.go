package textutil

import (
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func Backspace(te *widget.TextEdit) error {
	tc := te.TextCursor
	tc.BeginEdit()
	defer tc.EndEdit()

	var a, b int
	if tc.SelectionOn() {
		a, b = tc.SelectionIndexes()
		tc.SetSelectionOff()
	} else {
		b = tc.Index()
		rw := tc.RW()
		_, size, err := rw.ReadLastRuneAt(b)
		if err != nil {
			return err
		}
		a = b - size
	}
	if err := tc.RW().Delete(a, b-a); err != nil {
		return err
	}
	tc.SetIndex(a)
	return nil
}
