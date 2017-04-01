package cmdutil

import (
	"strings"

	"github.com/jmigpin/editor/drawutil"
	"github.com/jmigpin/editor/edit/toolbardata"
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
	tbsd := erow.ToolbarSD()
	ta := row.Toolbar
	found := false
	var part *toolbardata.Part
	for _, p := range tbsd.Parts {
		if len(p.Args) > 0 && p.Args[0].Str == "Find" {
			found = true
			part = p
			break
		}
	}

	if !found {
		// insert find cmd
		ta.EditOpen()
		ta.EditInsert(len(ta.Str()), " | Find ")
		ta.SetSelectionOn(false)
		a := len(ta.Str())
		if searchStr != "" {
			ta.SetSelectionOn(true)
			ta.SetSelectionIndex(a)
			ta.EditInsert(a, searchStr)
			a = len(ta.Str())
		}
		ta.SetCursorIndex(a + len(searchStr))
		ta.EditClose()
	} else {
		ta.EditOpen()
		// select current find cmd string
		a := part.Start + part.Args[0].End
		b := part.End
		if a == b {
			ta.EditInsert(a, " ")
			a++
			b++
		} else {
			a++
			ta.SetSelectionOn(true)
			ta.SetSelectionIndex(a)
		}
		ta.SetCursorIndex(b)

		// replace current find cmd string with search str
		if searchStr != "" {
			ta.EditDelete(a, b)
			ta.EditInsert(a, searchStr)
			ta.SetSelectionOn(true)
			ta.SetSelectionIndex(a)
			ta.SetCursorIndex(a + len(searchStr))
		}
		ta.EditClose()
	}

	// warp pointer to toolbar close to "Find " text cmd to be able to click for run
	p := ta.IndexPoint(ta.CursorIndex())
	p2 := drawutil.Point266ToPoint(p)
	p3 := p2.Add(ta.C.Bounds.Min)
	p3.Y += ta.LineHeight().Round() / 2 // center of rune
	erow.Ed().UI().WarpPointer(&p3)
}
