package core

import (
	"bytes"
	"strconv"

	"github.com/jmigpin/editor/core/toolbarparser"
)

// Search/add the toolbar find command and warps the pointer to it.
func FindShortcut(erow *ERow) {
	if err := findShortcut2(erow); err != nil {
		erow.Ed.Error(err)
		return
	}

	// warp pointer to toolbar close to "Find " text cmd to be able to click for run
	tb := erow.Row.Toolbar
	p := tb.GetPoint(tb.TextCursor.Index())
	p.Y += tb.LineHeight() * 3 / 4 // center of rune
	erow.Ed.UI.WarpPointer(&p)
}

func findShortcut2(erow *ERow) error {
	// check if there is a selection in the textarea
	searchStr := []byte{}
	tatc := erow.Row.TextArea.TextCursor
	if tatc.SelectionOn() {
		s, err := tatc.Selection()
		if err != nil {
			return err
		}
		// don't use if selection has more then one line
		if !bytes.ContainsRune(s, '\n') {
			searchStr = s
			// quote if it has spaces
			if bytes.ContainsRune(searchStr, ' ') {
				searchStr = []byte(strconv.Quote(string(searchStr)))
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
	tc := erow.Row.Toolbar.TextCursor
	tc.BeginEdit()
	defer tc.EndEdit()

	if found {
		// select current find cmd string
		a := part.Args[0].End
		b := part.End
		if a == b {
			if err := tc.RW().Insert(a, []byte(" ")); err != nil {
				return err
			}
			a++
			b++
			tc.SetIndex(b)
		} else {
			a++
			tc.SetSelection(a, b)
		}

		// replace current find cmd string with search str
		if len(searchStr) != 0 {
			if err := tc.RW().Delete(a, b-a); err != nil {
				return err
			}
			if err := tc.RW().Insert(a, searchStr); err != nil {
				return err
			}
			tc.SetSelection(a, a+len(searchStr))
			tatc.SetSelectionOff() // deselect to allow to search full text
		}
	} else {
		// insert find cmd
		tbl := tb.TextCursor.RW().Len()
		find := " | Find "
		if err := tc.RW().Insert(tbl, []byte(find)); err != nil {
			return err
		}
		a := tbl + len(find)
		if len(searchStr) != 0 {
			if err := tc.RW().Insert(a, searchStr); err != nil {
				return err
			}
			tc.SetSelection(a, a+len(searchStr))
			tatc.SetSelectionOff() // deselect to allow to search full text
		} else {
			tc.SetSelectionOff()
			tc.SetIndex(a + len(searchStr))
		}
	}
	return nil
}
