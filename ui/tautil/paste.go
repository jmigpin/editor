package tautil

func PastePrimary(ta Texta) {
	pasteFn(ta, ta.RequestPrimaryPaste)
}
func PasteClipboard(ta Texta) {
	pasteFn(ta, ta.RequestClipboardPaste)
}
func pasteFn(ta Texta, fn func() (string, error)) {
	// The requestclipboardstring blocks while it communicates with the x server. The x server answer can only be handled if this procedure is not blocking the eventloop.
	go func() {
		str, err := fn()
		if err != nil {
			ta.Error(err)
			return
		}
		InsertString(ta, str)
		// inside a goroutine, need to request paint
		ta.RequestTreePaint()
	}()
}
