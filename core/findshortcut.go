package core

import (
	"bytes"
	"strconv"

	"github.com/jmigpin/editor/core/toolbarparser"
)

func CaptureSelection(erow *ERow) []byte {
	text := []byte{}
	ta := erow.Row.TextArea

	// check if there is a selection in the textarea
	if selection_text, ok := ta.EditCtx().Selection(); ok {
		// don't use if selection has more then one line
		if !bytes.ContainsRune(selection_text, '\n') {
			text = selection_text
			// quote if it has spaces
			if bytes.ContainsRune(text, ' ') {
				text = []byte(strconv.Quote(string(text)))
			}
		}
	}
	return text
}

func ToolbarMatch(erow *ERow, s string) (*toolbarparser.Part, bool) {
	// find cmd in toolbar string
	found := false
	var part *toolbarparser.Part
	for _, p := range erow.TbData.Parts {
		if len(p.Args) > 0 && p.Args[0].String() == s {
			found = true
			part = p
			// don't break, find the last one
		}
	}
	return part, found
}

func ToolbarInsertion(erow *ERow, part *toolbarparser.Part, capture []byte, found bool, command string) error {
	tb := erow.Row.Toolbar
	c := tb.Cursor()

	tb.BeginUndoGroup()
	defer tb.EndUndoGroup()

	if found {
		// select current find cmd string
		a := part.Args[0].End()
		b := part.End()
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
		if len(capture) != 0 {
			if err := tb.RW().OverwriteAt(a, b-a, capture); err != nil {
				return err
			}
			c.SetSelection(a, a+len(capture))
		}
	} else {
		// insert find cmd
		tbl := tb.RW().Max()
		find := " | " + command + " "
		if err := tb.RW().OverwriteAt(tbl, 0, []byte(find)); err != nil {
			return err
		}
		a := tbl + len(find)
		if len(capture) != 0 {
			if err := tb.RW().OverwriteAt(a, 0, capture); err != nil {
				return err
			}
			c.SetSelection(a, a+len(capture))
		} else {
			c.SetIndexSelectionOff(a + len(capture))
		}
	}
	return nil
}

// update toolbar with shortcut
func ToolbarUpdate(erow *ERow, strToMatch string) error {
	capture := CaptureSelection(erow)
	part, found := ToolbarMatch(erow, strToMatch)
	return ToolbarInsertion(erow, part, capture, found, strToMatch)
}

// Search/add the toolbar find command and warps the pointer to it.
func FindShortcut(erow *ERow) {
	if err := ToolbarUpdate(erow, "Find"); err != nil {
		erow.Ed.Error(err)
		return
	}

	// warp pointer to toolbar close to "Find " text cmd to be able to click for run
	tb := erow.Row.Toolbar
	p := tb.GetPoint(tb.CursorIndex())
	p.Y += tb.LineHeight() * 3 / 4 // center of rune
	erow.Ed.UI.WarpPointer(p)
}

// Search/add the toolbar Replace command and warps the pointer to it.
func ReplaceShortcut(erow *ERow) {
	if err := ToolbarUpdate(erow, "Replace"); err != nil {
		erow.Ed.Error(err)
		return
	}

	// warp pointer to toolbar close to added text
	tb := erow.Row.Toolbar
	p := tb.GetPoint(tb.CursorIndex())
	p.Y += tb.LineHeight() * 3 / 4 // center of rune
	erow.Ed.UI.WarpPointer(p)
}

// Search/add the toolbar NewFile command and warps the pointer to it.
func NewFileShortcut(erow *ERow) {
	if err := ToolbarUpdate(erow, "NewFile"); err != nil {
		erow.Ed.Error(err)
		return
	}

	// warp pointer to toolbar close to added text
	tb := erow.Row.Toolbar
	p := tb.GetPoint(tb.CursorIndex())
	p.Y += tb.LineHeight() * 3 / 4 // center of rune
	erow.Ed.UI.WarpPointer(p)
}
