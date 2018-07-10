package textutil

import "github.com/jmigpin/editor/util/uiutil/widget"

func RemoveLines(te *widget.TextEdit) error {
	tc := te.TextCursor
	tc.BeginEdit()
	defer tc.EndEdit()

	a, b, _, err := tc.LinesIndexes()
	if err != nil {
		return err
	}
	if err := tc.RW().Delete(a, b-a); err != nil {
		return err
	}
	tc.SetSelectionOff()
	tc.SetIndex(a)
	return nil
}
