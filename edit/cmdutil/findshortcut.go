package cmdutil

import (
	"github.com/jmigpin/editor/drawutil"
	"github.com/jmigpin/editor/edit/toolbardata"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/ui/tautil"
)

// Search for the find command in the toolbar and warps the pointer to it. Adds the command to the toolbar if not present.
func FindShortcut(ed Editori, row *ui.Row) {
	// check if there is a selection in the textarea
	searchStr := ""
	if row.TextArea.SelectionOn() {
		a, b := tautil.SelectionStringIndexes(row.TextArea)
		searchStr = row.TextArea.Str()[a:b]
	}

	// find cmd in toolbar string
	tsd := ed.RowToolbarStringData(row)
	ta := row.Toolbar
	found := false
	var part *toolbardata.Part
	for _, p := range tsd.Parts {
		if len(p.Args) > 0 && p.Args[0].Str == "Find" {
			found = true
			part = p
			break
		}
	}
	if !found || searchStr != "" {
		// insert find cmd
		ta.EditInsert(len(ta.Str()), " | Find "+searchStr)
		ta.EditDone()
		ta.SetSelectionOn(false)
		ta.SetCursorIndex(len(ta.Str()))
	} else if found {
		if len(part.Args) == 1 {
			// no other args
			a := part.Start + part.Args[0].End
			b := part.End
			if a == b {
				// insert a space
				ta.EditInsert(a, " ")
				ta.EditDone()
			}
			ta.SetSelectionOn(false)
			ta.SetCursorIndex(a + 1)
		} else {
			// select arg string
			a := part.Start + part.Args[0].End + 1
			b := part.Start + part.Args[len(part.Args)-1].End
			ta.SetSelectionOn(true)
			ta.SetSelectionIndex(a)
			ta.SetCursorIndex(b)
		}
	}

	// warp pointer to toolbar close to "Find " text cmd to be able to click for run
	p := ta.IndexPoint266(ta.CursorIndex())
	p2 := drawutil.Point266ToPoint(p)
	p3 := p2.Add(ta.C.Bounds.Min)
	p3.Y += ta.LineHeight().Round() / 2 // center of rune
	ed.UI().WarpPointer(&p3)
}
