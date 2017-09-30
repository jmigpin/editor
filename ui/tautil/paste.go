package tautil

import "log"

func PastePrimary(ta Texta) {
	pasteFn(ta, ta.RequestPrimaryPaste)
}
func PasteClipboard(ta Texta) {
	pasteFn(ta, ta.RequestClipboardPaste)
}
func pasteFn(ta Texta, fn func() (string, error)) {
	// The request blocks while it communicates with the x server.
	// The x server answer can only be handled if this procedure is not blocking the eventloop.
	go func() {
		str, err := fn()
		if err != nil {
			// TODO
			log.Print(err)
			return
		}
		ta.InsertStringAsync(str)
	}()
}
