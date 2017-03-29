package tautil

func Copy(ta Texta) {
	if !ta.SelectionOn() {
		return
	}
	a, b := SelectionStringIndexes(ta)
	s := ta.Str()[a:b]
	ta.SetCopyClipboard(s)
}
