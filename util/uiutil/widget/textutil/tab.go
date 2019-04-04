package textutil

import (
	"bytes"

	"github.com/jmigpin/editor/util/uiutil/widget"
)

func TabRight(te *widget.TextEdit) error {
	tc := te.TextCursor

	if !tc.SelectionOn() {
		return InsertString(te, "\t")
	}

	tc.BeginEdit()
	defer tc.EndEdit()

	a, b, newline, err := tc.LinesIndexes()
	if err != nil {
		return err
	}

	// insert at lines start
	for i := a; i < b; {
		if err := tc.RW().Insert(i, []byte{'\t'}); err != nil {
			return err
		}
		b += 1 // size of \t

		u, _, err := te.LineEndIndex(i)
		if err != nil {
			return err
		}
		i = u
	}

	// cursor index without the newline
	if newline {
		b--
	}

	tc.SetSelection(a, b)
	return nil
}

func TabLeft(te *widget.TextEdit) error {
	tc := te.TextCursor

	a, b, newline, err := tc.LinesIndexes()
	if err != nil {
		return err
	}

	tc.BeginEdit()
	defer tc.EndEdit()

	// remove from lines start
	altered := false
	for i := a; i < b; {
		s, err := tc.RW().ReadNCopyAt(i, 1)
		if err != nil {
			return err
		}
		if bytes.ContainsAny(s, "\t ") {
			altered = true
			if err := tc.RW().Delete(i, 1); err != nil {
				return err
			}
			b -= 1 // 1 is length of '\t' or ' '
		}

		u, _, err := te.LineEndIndex(i)
		if err != nil {
			return err
		}
		i = u
	}

	// skip making the selection
	if !altered {
		return nil
	}

	// cursor index without the newline
	if newline {
		b--
	}

	tc.SetSelection(a, b)
	return nil
}
