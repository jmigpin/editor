package textutil

import (
	"github.com/jmigpin/editor/core/parseutil"
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func SelectWord(te *widget.TextEdit) error {
	tc := te.TextCursor

	// index rune
	ci := tc.Index()
	ru, _, err := tc.RW().ReadRuneAt(ci)
	if err != nil {
		return err
	}

	var index int
	var word []byte
	if !parseutil.IsWordRune(ru) {
		// select just the index rune
		index = ci
		word = []byte(string(ru))
	} else {
		// select word at index
		w, i, err := parseutil.WordAtIndex(tc.RW(), ci, 100)
		if err != nil {
			return err
		}

		index = i
		word = w
	}

	tc.SetSelection(index, index+len(word))

	// set primary copy
	if tc.SelectionOn() {
		s, err := tc.Selection()
		if err == nil {
			te.SetCPCopy(event.CPIPrimary, string(s))
		}
	}

	return nil
}
