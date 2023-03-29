package core

import (
	"bytes"
	"strconv"

	"github.com/jmigpin/editor/core/toolbarparser"
)

func AddFindShortcut(erow *ERow) {
	updateToolbarPartCmd(erow, "Find")
}
func AddReplaceShortcut(erow *ERow) {
	updateToolbarPartCmd(erow, "Replace")
}
func AddNewFileShortcut(erow *ERow) {
	updateToolbarPartCmd(erow, "NewFile")
}
func AddReloadShortcut(erow *ERow) {
	updateToolbarPartCmd2(erow, "Reload", true)
}

//----------

// Search/add toolbar command and warps the pointer to it. Also inserts selected text as argument if available.
func updateToolbarPartCmd(erow *ERow, cmd string) {
	updateToolbarPartCmd2(erow, cmd, false)
}
func updateToolbarPartCmd2(erow *ERow, cmd string, ignoreSelection bool) {
	if err := updateToolbarPartCmd3(erow, cmd, ignoreSelection); err != nil {
		erow.Ed.Error(err)
	}
}
func updateToolbarPartCmd3(erow *ERow, cmd string, ignoreSelection bool) error {
	// modify toolbar text
	arg := ""
	if !ignoreSelection {
		arg = erowTextSelection(erow)
	}
	res := toolbarparser.UpdateOrInsertPartCmd(&erow.TbData, cmd, arg)

	// update toolbar text
	tb := erow.Row.Toolbar
	if err := tb.SetStr(res.S); err != nil {
		return err
	}

	// update cursor position
	c := tb.Cursor()
	c.SetIndex(res.Pos)
	c.UpdateSelection(true, res.End)

	// warp pointer to toolbar close to added text
	p := tb.GetPoint(tb.CursorIndex())
	p.Y += tb.LineHeight() * 3 / 4 // center of rune
	erow.Ed.UI.WarpPointer(p)

	return nil
}
func erowTextSelection(erow *ERow) string {
	ta := erow.Row.TextArea
	text := []byte{}
	// check if there is a selection in the textarea
	if sel, ok := ta.EditCtx().Selection(); ok {
		// don't use if selection has more then one line
		if !bytes.ContainsRune(sel, '\n') {
			text = sel
			// quote if it has spaces
			if bytes.ContainsRune(text, ' ') {
				text = []byte(strconv.Quote(string(text)))
			}
		}
	}
	return string(text)
}
