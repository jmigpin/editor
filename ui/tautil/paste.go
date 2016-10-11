package tautil

func Paste(ta Texta) {
	go func() {
		str, err := ta.RequestClipboardString()
		if err != nil {
			ta.Error(err)
			return
		}
		if str == "" {
			return
		}

		text := ta.Text()

		if ta.SelectionOn() {
			a, b, ok := selectionStringIndexes(ta)
			if !ok {
				return
			}
			text = text[:a] + text[b:] // remove selection
			ta.SetCursorIndex(a)
		}

		i := ta.CursorIndex()
		text = text[:i] + str + text[i:] // insert

		ta.SetText(text)

		ta.SetSelectionOn(false)
		ta.SetCursorIndex(ta.CursorIndex() + len(str))
		ta.RequestTreePaint()
	}()
}
