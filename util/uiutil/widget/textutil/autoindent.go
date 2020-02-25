package textutil

import (
	"errors"
	"io"
	"unicode"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func AutoIndent(te *widget.TextEdit) error {
	tc := te.TextCursor
	tc.BeginEdit()
	defer tc.EndEdit()

	ci := tc.Index()
	i, err := te.LineStartIndex(ci)
	if err != nil {
		return err
	}

	rd := iorw.NewLimitedReader(tc.RW(), i, ci)
	j, _, err := iorw.IndexFunc(rd, i, false, unicode.IsSpace)
	if err != nil {
		if errors.Is(err, io.EOF) {
			j = ci // all spaces up to ci
		} else {
			return err
		}
	}

	// string to insert
	s, err := tc.RW().ReadNCopyAt(i, j-i)
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
