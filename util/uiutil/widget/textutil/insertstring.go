package textutil

import "github.com/jmigpin/editor/util/uiutil/widget"

func InsertString(te *widget.TextEdit, s string) error {
	tc := te.TextCursor
	tc.BeginEdit()
	defer tc.EndEdit()

	if tc.SelectionOn() {
		// remove selection
		a, b := tc.SelectionIndexes()
		if err := tc.RW().Delete(a, b-a); err != nil {
			return err
		}
		tc.SetSelectionOff()
		tc.SetIndex(a)
	}
	// insert
	if err := tc.RW().Insert(tc.Index(), []byte(s)); err != nil {
		return err
	}
	tc.SetIndex(tc.Index() + len(s))
	return nil
}
