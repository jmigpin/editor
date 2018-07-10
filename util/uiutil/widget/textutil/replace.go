package textutil

import (
	"github.com/jmigpin/editor/util/iout"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func Replace(te *widget.TextEdit, old, new string) (bool, error) {
	if old == "" {
		return false, nil
	}

	tc := te.TextCursor
	tc.BeginEdit()
	defer tc.EndEdit()

	oldb := []byte(old)
	newb := []byte(new)

	var a, b int
	if tc.SelectionOn() {
		a, b = tc.SelectionIndexes()
	} else {
		a = 0
		b = tc.RW().Len()
	}

	ci, replaced, err := replace2(tc, oldb, newb, a, b)
	if err == nil {
		tc.SetIndex(ci)
	}

	return replaced, err
}

func replace2(tc *widget.TextCursor, oldb, newb []byte, a, b int) (int, bool, error) {
	ci := tc.Index()
	replaced := false
	for a < b {
		i, err := iout.Index(tc.RW(), a, b-a, oldb, false)
		if err != nil {
			return ci, replaced, err
		}
		if i < 0 {
			return ci, replaced, nil
		}
		if err := tc.RW().Delete(i, len(oldb)); err != nil {
			return ci, replaced, err
		}
		if err := tc.RW().Insert(i, newb); err != nil {
			return ci, replaced, err
		}
		replaced = true
		d := -len(oldb) + len(newb)
		b += d
		a = i + len(newb)

		if i < ci {
			ci += d
			if ci < i {
				ci = i
			}
		}
	}
	return ci, replaced, nil
}
