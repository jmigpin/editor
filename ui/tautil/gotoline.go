package tautil

import (
	"strconv"
)

func GotoLine(ta Texta, str string) {
	n, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return
	}
	_ = GotoLineNum(ta, int(n))
}

// Returns true if index was found.
func GotoLineNum(ta Texta, n int) bool {
	if n <= 1 {
		gotoIndex(ta, 0)
		return true
	}
	for ri, ru := range ta.Str() {
		if ru == '\n' {
			n--
			if n <= 1 {
				gotoIndex(ta, ri+1) // +1 is lenght of '\n'
				return true
			}
		}
	}
	return false
}
func gotoIndex(ta Texta, index int) {
	ta.SetSelectionOff()
	ta.SetCursorIndex(index)
	ta.MakeIndexVisibleAtCenter(index)
	ta.WarpPointerToIndexIfVisible(index)
}
