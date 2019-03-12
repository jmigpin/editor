package textutil

import (
	"bytes"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func Find(te *widget.TextEdit, str string) (bool, error) {
	if str == "" {
		return false, nil
	}

	// ignore case
	strb := bytes.ToLower([]byte(str))

	tc := te.TextCursor
	i, err := find2(tc, strb)
	if err != nil {
		return false, err
	}
	if i >= 0 {
		tc.SetSelection(i, i+len(str)) // cursor at end to allow searching next
		te.MakeIndexVisible(i)
		return true, nil
	}
	return false, nil
}

func find2(tc *widget.TextCursor, s []byte) (int, error) {
	ci := tc.Index()
	l := tc.RW().Len()

	// index to end
	i, err := iorw.Index(tc.RW(), ci, l, s, true)
	if err != nil || i >= 0 {
		return i, err
	}

	// start to index
	w := ci + len(s) - 1
	if w > l {
		w = l
	}
	return iorw.Index(tc.RW(), 0, w, s, true)
}
