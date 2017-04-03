package tautil

func SelectAll(ta Texta) {
	ta.SetSelection(0, len(ta.Str()))
}
