package tautil

import "strings"

func Replace(ta Texta, old, new string) {
	var t string
	if ta.SelectionOn() {
		a, b, ok := selectionStringIndexes(ta)
		if !ok {
			return
		}
		s := ta.Text()[a:b]
		s2 := strings.Replace(s, old, new, -1)
		t = ta.Text()[:a] + s2 + ta.Text()[b:]
		ta.SetCursorIndex(a + len(s2))
	} else {
		t = strings.Replace(ta.Text(), old, new, -1)
	}
	ta.SetText(t)
}
