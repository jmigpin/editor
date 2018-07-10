package textutil

import (
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func DuplicateLines(te *widget.TextEdit) error {
	tc := te.TextCursor
	tc.BeginEdit()
	defer tc.EndEdit()

	a, b, newline, err := tc.LinesIndexes()
	if err != nil {
		return err
	}

	s, err := tc.RW().ReadNAt(a, b-a)
	if err != nil {
		return err
	}

	c := b
	if !newline {
		s = append([]byte{'\n'}, s...)
		c++
	}

	if err := tc.RW().Insert(b, s); err != nil {
		return err
	}

	// cursor index without the newline
	d := b + len(s)
	if newline && len(s) > 0 && s[len(s)-1] == '\n' {
		d--
	}

	tc.SetSelection(c, d)
	return nil
}
