package tautil

import "github.com/jmigpin/editor/uiutil/event"

func SelectLine(ta Texta) {
	ta.SetSelectionOff()
	a, b, _ := linesStringIndexes(ta)
	ta.SetSelection(a, b)

	// set primary copy
	if ta.SelectionOn() {
		a, b := SelectionStringIndexes(ta)
		s := ta.Str()[a:b]
		err := ta.SetCPCopy(event.PrimaryCPI, s)
		if err != nil {
			ta.Error(err)
		}
	}
}
