package tautil

func SelectLine(ta Texta) {
	ta.SetSelectionOff()
	a, b, _ := linesStringIndexes(ta)
	ta.SetSelection(a, b)

	// set primary copy
	if ta.SelectionOn() {
		a, b := SelectionStringIndexes(ta)
		s := ta.Str()[a:b]
		ta.SetPrimaryCopy(s)
	}
}
