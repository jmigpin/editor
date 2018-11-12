package textutil

import (
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func MoveLineUp(te *widget.TextEdit) error {
	tc := te.TextCursor
	tc.BeginEdit()
	defer tc.EndEdit()

	a, b, newline, err := tc.LinesIndexes()
	if err != nil {
		return err
	}
	// already at the first line
	if a == 0 {
		return nil
	}

	s, err := tc.RW().ReadNAt(a, b-a)
	if err != nil {
		return err
	}

	if err := tc.RW().Delete(a, b-a); err != nil {
		return err
	}

	a2, err := iout.LineStartIndex(tc.RW(), a-1) // start of previous line, -1 is size of '\n'
	if err != nil {
		return err
	}

	// remove newline to honor the moving line
	if !newline {
		if err := tc.RW().Delete(a-1, 1); err != nil {
			return err
		}
		s = append(s, '\n')
	}

	if err := tc.RW().Insert(a2, s); err != nil {
		return err
	}

	if tc.SelectionOn() {
		b2 := a2 + len(s)
		_, size, err := tc.RW().ReadLastRuneAt(b2)
		if err != nil {
			return nil
		}
		tc.SetSelection(a2, b2-size)
	} else {
		// position cursor at same position
		tc.SetIndex(tc.Index() - (a - a2))
	}
	return nil
}

func MoveLineDown(te *widget.TextEdit) error {
	tc := te.TextCursor
	tc.BeginEdit()
	defer tc.EndEdit()

	a, b, newline, err := tc.LinesIndexes()
	if err != nil {
		return err
	}
	// already at the last line
	if !newline && b == tc.RW().Len() {
		return nil
	}

	// keep copy of the moving line
	s, err := tc.RW().ReadNAt(a, b-a)
	if err != nil {
		return err
	}

	// delete moving line
	if err := tc.RW().Delete(a, b-a); err != nil {
		return err
	}

	// line end of the line below
	a2, newline, err := iout.LineEndIndex(tc.RW(), a)
	if err != nil {
		return err
	}

	// remove newline
	if !newline {
		// remove newline
		s = s[:len(s)-1]
		// insert newline
		if err := tc.RW().Insert(a2, []byte{'\n'}); err != nil {
			return err
		}
		a2 += 1 // 1 is '\n' added to s before insertion
	}

	if err := tc.RW().Insert(a2, s); err != nil {
		return err
	}

	if tc.SelectionOn() {
		b2 := a2 + len(s)
		// don't select newline
		if newline {
			_, size, err := tc.RW().ReadLastRuneAt(b2)
			if err != nil {
				return nil
			}
			b2 -= size
		}
		tc.SetSelection(a2, b2)
	} else {
		// position cursor at same position
		tc.SetIndex(tc.Index() + (a2 - a))
	}
	return nil
}
