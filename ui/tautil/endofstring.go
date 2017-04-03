package tautil

func EndOfString(ta Texta, sel bool) {
	i := len(ta.Str())
	updateSelection(ta, sel, i)
}
