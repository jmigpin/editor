package tautil

import (
	"strconv"
)

func GotoLine(ta Texta, str string) {
	line, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return
	}
	if line <= 1 {
		gotoIndex(ta, 0)
		return
	}
	for ri, ru := range ta.Str() {
		if ru == '\n' {
			line--
			if line <= 1 {
				gotoIndex(ta, ri+1) // +1 is lenght of '\n'
				return
			}
		}
	}
}
func gotoIndex(ta Texta, index int) {
	ta.SetCursorIndex(index)
	ta.MakeIndexVisible(index)
}
