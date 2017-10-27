package cmdutil

import (
	"strings"

	"github.com/jmigpin/editor/core/toolbardata"
	"github.com/jmigpin/editor/ui/tautil"
)

// Search/add the toolbar find command and warps the pointer to it.
func FindShortcut(erow ERower) {
	row := erow.Row()

	// check if there is a selection in the textarea
	searchStr := ""
	if row.TextArea.SelectionOn() {
		a, b := tautil.SelectionStringIndexes(row.TextArea)
		searchStr = row.TextArea.Str()[a:b]
		// if more then one line - no search string
		for _, ru := range searchStr {
			if ru == '\n' {
				searchStr = "" //searchStr[:i]
				break
			}
		}

		searchStr = strings.TrimSpace(searchStr)
	}

	// find cmd in toolbar string
	td := erow.ToolbarData()
	ta := row.Toolbar
	found := false
	var part *toolbardata.Part
	for _, p := range td.Parts {
		if len(p.Args) > 0 && p.Args[0].Str == "Find" {
			found = true
			part = p
			break
		}
	}

	ta.EditOpen()
	defer ta.EditCloseAfterSetCursor()

	if found {
		// select current find cmd string
		a := part.Args[0].E
		b := part.E
		if a == b {
			ta.EditInsert(a, " ")
			a++
			b++
			ta.SetCursorIndex(b)
		} else {
			a++
			ta.SetSelection(a, b)
		}

		// replace current find cmd string with search str
		if searchStr != "" {
			ta.EditDelete(a, b)
			ta.EditInsert(a, searchStr)
			ta.SetSelection(a, a+len(searchStr))
		}
	} else {
		// insert find cmd
		ta.EditInsert(len(ta.Str()), " | Find ")
		a := len(ta.Str())
		if searchStr != "" {
			ta.EditInsert(a, searchStr)
			ta.SetSelection(a, a+len(searchStr))
		} else {
			ta.SetSelectionOff()
			ta.SetCursorIndex(a + len(searchStr))
		}
	}

	// warp pointer to toolbar close to "Find " text cmd to be able to click for run
	p := ta.GetPoint(ta.CursorIndex())
	p3 := p.Add(ta.Bounds().Min)
	p3.Y += ta.LineHeight() / 2 // center of rune
	erow.Ed().UI().WarpPointer(&p3)
}
