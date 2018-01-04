package tautil

import "github.com/jmigpin/editor/util/uiutil/event"

func Copy(ta Texta) {
	if !ta.SelectionOn() {
		return
	}
	a, b := SelectionStringIndexes(ta)
	s := ta.Str()[a:b]
	err := ta.SetCPCopy(event.ClipboardCPI, s)
	if err != nil {
		ta.Error(err)
	}
}
