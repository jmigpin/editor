package tautil

func SelectLine(ta Texta) {
	ta.SetSelectionOff()
	a, b, _ := linesStringIndexes(ta)
	ta.SetSelection(a, b)
}
