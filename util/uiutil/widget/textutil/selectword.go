package textutil

import (
	"github.com/jmigpin/editor/util/iout/iorw"
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
	if !iorw.IsWordRune(ru) {
		// select just the index rune
		index = ci
		word = []byte(string(ru))
	} else {
		// select word at index
		rd := te.LimitedReaderPad(ci)
		w, i, err := iorw.WordAtIndex(rd, ci)
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
