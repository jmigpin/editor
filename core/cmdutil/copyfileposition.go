package cmdutil

import "fmt"

func CopyFilePosition(ed Editorer, erow ERower) {
	if !erow.IsRegular() {
		ed.Errorf("not a regular file")
		return
	}

	ta := erow.Row().TextArea
	ci := ta.CursorIndex()
	line, lineStart := 0, 0
	for ri, ru := range ta.Str()[:ci] {
		if ri >= ci {
			break
		}
		if ru == '\n' {
			line++
			lineStart = ri
		}
	}
	col := ci - lineStart
	line++
	s := fmt.Sprintf("%v:%v:%v", erow.Filename(), line, col)
	ta.SetClipboardCopy(s)
}
