package tautil

import "github.com/jmigpin/editor/util/uiutil/event"

func Paste(ta Texta, i event.CopyPasteIndex) {
	// The request blocks while it communicates with the x server.
	// The x server answer can only be handled if this procedure is not blocking the eventloop.
	go func() {
		str, err := ta.GetCPPaste(i)
		if err != nil {
			ta.Error(err)
			return
		}
		ta.InsertStringAsync(str)
	}()
}
