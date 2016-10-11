package tautil

func RemoveLines(ta Texta) {
	a, b, ok := linesStringIndexes(ta)
	if !ok {
		return
	}
	ta.SetText(ta.Text()[:a] + ta.Text()[b:]) // remove text
	ta.SetSelectionOn(false)
	ta.SetCursorIndex(a)
}
