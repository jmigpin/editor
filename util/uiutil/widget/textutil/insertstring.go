package textutil

import "github.com/jmigpin/editor/util/uiutil/widget"

func InsertString(te *widget.TextEdit, s string) error {
	tc := te.TextCursor
	tc.BeginEdit()
	defer tc.EndEdit()

	n := 0
	ci := tc.Index()
	if tc.SelectionOn() {
		a, b := tc.SelectionIndexes()
		n = b - a
		ci = a
		tc.SetSelectionOff()
	}
	if err := tc.RW().Overwrite(ci, n, []byte(s)); err != nil {
		return err
	}
	tc.SetIndex(ci + len(s))
	return nil
}
