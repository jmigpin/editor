package textutil

import (
	"io"
	"unicode"

	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func AutoIndent(te *widget.TextEdit) error {
	tc := te.TextCursor
	tc.BeginEdit()
	defer tc.EndEdit()

	ci := tc.Index()
	i, err := iout.LineStartIndex(tc.RW(), ci)
	if err != nil {
		return err
	}

	j, _, err := iout.IndexFunc(tc.RW(), i, ci-i, false, unicode.IsSpace)
	if err != nil {
		if err == io.EOF {
			// full line of spaces, indent to ci
			j = ci
		} else if err == iout.ErrLimitReached {
			// all spaces up to ci
			j = ci
		} else {
			return err
		}
	}

	// string to insert
	s, err := tc.RW().ReadNAt(i, j-i)
	if err != nil {
		return err
	}
	s2 := append([]byte{'\n'}, s...)

	// remove selection
	if tc.SelectionOn() {
		a, b := tc.SelectionIndexes()
		if err := tc.RW().Delete(a, b-a); err != nil {
			return err
		}
		tc.SetSelectionOff()
		tc.SetIndex(a)
	}

	// insert
	ci = tc.Index()
	if err := tc.RW().Insert(ci, s2); err != nil {
		return err
	}
	tc.SetIndex(ci + len(s2))
	return nil
}
