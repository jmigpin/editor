package edit

import (
	"jmigpin/editor/drawutil"
	"jmigpin/editor/edit/toolbar"
	"jmigpin/editor/ui"
)

// Search for the find command in the toolbar and warps the pointer to it. Adds the command to the toolbar if not present.
func quickFindShortcut(ed *Editor, row *ui.Row) {
	tsd := ed.rowToolbarStringData(row)
	ta := row.Toolbar
	found := false
	var part *toolbar.Part
	for _, p := range tsd.Parts {
		if len(p.Args) > 0 && p.Args[0].Str == "Find" {
			found = true
			part = p
			break
		}
	}
	if found {
		if len(part.Args) == 1 {
			// no other args
			a := part.Start + part.Args[0].End
			b := part.End
			if a == b {
				// insert a space
				ta.SetText(ta.Text()[:a] + " " + ta.Text()[a:])
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
	} else {
		// insert find cmd
		ta.SetText(ta.Text() + " | Find ")
		ta.SetSelectionOn(false)
		ta.SetCursorIndex(len(ta.Text()))
	}

	// warp pointer to toolbar close to "Find " text cmd to be able to click for run
	p := ta.IndexPoint266(ta.CursorIndex())
	p2 := drawutil.Point266ToPoint(p)
	p3 := p2.Add(ta.Area.Min)
	p3.Y += ta.LineHeight().Round() / 2 // center of rune
	ta.UI.WarpPointer(&p3)
}
