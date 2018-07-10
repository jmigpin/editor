package textutil

import (
	"github.com/jmigpin/editor/util/uiutil/event"
	"github.com/jmigpin/editor/util/uiutil/widget"
)

func Copy(te *widget.TextEdit) error {
	tc := te.TextCursor
	if !tc.SelectionOn() {
		return nil
	}
	s, err := tc.Selection()
	if err != nil {
		return err
	}
	te.SetCPCopy(event.CPIClipboard, string(s))
	return nil
}
