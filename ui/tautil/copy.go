package tautil

func Copy(ta Texta) {
	if !ta.SelectionOn() {
		return
	}
	a, b, ok := selectionStringIndexes(ta)
	if !ok {
		return
	}
	s := ta.Str()[a:b]
	ta.SetClipboardString(s)
}
