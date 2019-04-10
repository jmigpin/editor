package textutil

import (
	"github.com/jmigpin/editor/util/iout/iorw"
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
		a = tc.RW().Min()
		b = tc.RW().Max()
	}

	ci, replaced, err := replace2(te, oldb, newb, a, b)
	if err == nil {
		tc.SetIndex(ci)
	}

	return replaced, err
}

func replace2(te *widget.TextEdit, oldb, newb []byte, a, b int) (int, bool, error) {
	tc := te.TextCursor

	ci := tc.Index()
	replaced := false
	for a < b {
		rd := iorw.NewLimitedReaderLen(tc.RW(), a, b-a)
		i, err := iorw.Index(rd, a, oldb, false)
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
