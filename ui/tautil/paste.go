package tautil

func Paste(ta Texta) {
	// The requestclipboard string blocks when it needs to communicate with the x server - hence using a go routine. An alternative would be to have it require a callback.
	go func() {
		str, err := ta.RequestClipboardString()
		if err != nil {
			ta.Error(err)
			return
		}
		if str == "" {
			return
		}

		if ta.SelectionOn() {
			a, b, ok := selectionStringIndexes(ta)
			if !ok {
				return
			}

			ta.EditRemove(a, b)
			ta.SetCursorIndex(a)
		}

		ta.EditInsert(ta.CursorIndex(), str)
		ta.EditDone()

		ta.SetSelectionOn(false)
		ta.SetCursorIndex(ta.CursorIndex() + len(str))
		ta.RequestTreePaint()
	}()
}
