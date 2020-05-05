package core

import (
	"bytes"
	"strconv"

	"github.com/jmigpin/editor/v2/core/toolbarparser"
)

// Search/add the toolbar find command and warps the pointer to it.
func FindShortcut(erow *ERow) {
	if err := findShortcut2(erow); err != nil {
		erow.Ed.Error(err)
		return
	}

	// warp pointer to toolbar close to "Find " text cmd to be able to click for run
	tb := erow.Row.Toolbar
	p := tb.GetPoint(tb.CursorIndex())
	p.Y += tb.LineHeight() * 3 / 4 // center of rune
	erow.Ed.UI.WarpPointer(p)
}

func findShortcut2(erow *ERow) error {
	// check if there is a selection in the textarea
	searchB := []byte{}
	ta := erow.Row.TextArea
	if b, ok := ta.EditCtx().Selection(); ok {
		// don't use if selection has more then one line
		if !bytes.ContainsRune(b, '\n') {
			searchB = b
			// quote if it has spaces
			if bytes.ContainsRune(searchB, ' ') {
				searchB = []byte(strconv.Quote(string(searchB)))
			}
		}
	}

	// find cmd in toolbar string
	found := false
	var part *toolbarparser.Part
	for _, p := range erow.TbData.Parts {
		if len(p.Args) > 0 && p.Args[0].Str() == "Find" {
			found = true
			part = p
			// don't break, find the last one
		}
	}

	tb := erow.Row.Toolbar
	c := tb.Cursor()

	tb.BeginUndoGroup()
	defer tb.EndUndoGroup()

	if found {
		// select current find cmd string
		a := part.Args[0].End
		b := part.End
		if a == b {
			if err := tb.RW().OverwriteAt(a, 0, []byte(" ")); err != nil {
				return err
			}
			a++
			b++
			c.SetIndex(b)
		} else {
			a++
			c.SetSelection(a, b)
		}

		// replace current find cmd string with search str
		if len(searchB) != 0 {
			if err := tb.RW().OverwriteAt(a, b-a, searchB); err != nil {
				return err
			}
			c.SetSelection(a, a+len(searchB))
		}
	} else {
		// insert find cmd
		tbl := tb.RW().Max()
		find := " | Find "
		if err := tb.RW().OverwriteAt(tbl, 0, []byte(find)); err != nil {
			return err
		}
		a := tbl + len(find)
		if len(searchB) != 0 {
			if err := tb.RW().OverwriteAt(a, 0, searchB); err != nil {
				return err
			}
			c.SetSelection(a, a+len(searchB))
		} else {
			c.SetIndexSelectionOff(a + len(searchB))
		}
	}
	return nil
}
