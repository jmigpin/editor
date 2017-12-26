package tautil

import "github.com/jmigpin/editor/uiutil/event"

func Cut(ta Texta) {
	if !ta.SelectionOn() {
		return
	}
	a, b := SelectionStringIndexes(ta)

	err := ta.SetCPCopy(event.ClipboardCPI, ta.Str()[a:b])
	if err != nil {
		ta.Error(err)
	}

	ta.EditOpen()
	defer ta.EditCloseAfterSetCursor()
	ta.EditDelete(a, b)
	ta.SetSelectionOff()
	ta.SetCursorIndex(a)
}
