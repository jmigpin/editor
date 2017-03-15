package tautil

func Paste(ta Texta) {
	// The requestclipboardstring blocks while it communicates with the x server. The x server answer can only be handled if this procedure is not blocking the eventloop.
	go func() {
		str, err := ta.RequestClipboardString()
		if err != nil {
			ta.Error(err)
			return
		}
		if str == "" {
			return
		}

		ta.EditOpen()
		if ta.SelectionOn() {
			a, b := SelectionStringIndexes(ta)
			ta.EditDelete(a, b)
			ta.SetCursorIndex(a)
		}
		ta.EditInsert(ta.CursorIndex(), str)
		ta.EditClose()

		ta.SetSelectionOn(false)
		ta.SetCursorIndex(ta.CursorIndex() + len(str))
		// inside a goroutine, need to request paint
		ta.RequestTreePaint()
	}()
}
