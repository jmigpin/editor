package tautil

func DupLines(ta Texta) {
	a, b, ok := linesStringIndexes(ta)
	if !ok {
		return
	}
	t := ta.Text()[a:b]
	ta.SetText(ta.Text()[:b] + t + ta.Text()[b:]) // insert text
	ta.SetSelectionOn(true)
	ta.SetSelectionIndex(b)

	_, b2, ok := PreviousRuneIndex(t, len(t))
	if !ok {
		panic("!")
	}
	b += b2
	ta.SetCursorIndex(b)
}
