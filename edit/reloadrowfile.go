package edit

import (
	"fmt"
	"io/ioutil"
	"jmigpin/editor/edit/toolbar"
	"jmigpin/editor/ui"
)

func reloadRowsFiles(ed *Editor) {
	for _, c := range ed.ui.Layout.Cols.Cols {
		for _, r := range c.Rows {
			reloadRowFile2(ed, r, true)
		}
	}
}
func reloadRowFile(ed *Editor, row *ui.Row) {
	reloadRowFile2(ed, row, false)
}
func reloadRowFile2(ed *Editor, row *ui.Row, tolerant bool) {
	tsd := toolbar.NewStringData(row.Toolbar.Text())
	filename, ok := tsd.FilenameTag()
	if !ok {
		if !tolerant {
			ed.Error(fmt.Errorf("row has no filename"))
		}
		return
	}
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		ed.Error(err)
		return
	}
	row.TextArea.SetText(string(b))
	row.TextArea.SetSelectionOn(false)
	row.Square.SetDirty(false)
	row.Square.SetCold(false)
}
