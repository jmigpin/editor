package edit

import (
	"io/ioutil"
	"jmigpin/editor/edit/toolbar"
	"jmigpin/editor/ui"
	"os"
)

// Opens directories (empty row with d set) or filenames.
func openPathAtCol(ed *Editor, path string, col *ui.Column) (*ui.Row, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		// always open a new row, even if other exists
		dir := path
		row := col.NewRow()
		row.Toolbar.SetText("d:" + dir)
		return row, nil
	}
	filename := path
	// it's a file already in a row
	row, ok := ed.findFilenameRow(filename)
	if ok {
		return row, nil
	}
	// it's file to open
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	// new row
	row = col.NewRow()
	row.Toolbar.SetText("f:" + toolbar.ReplaceHomeVar(filename))
	row.TextArea.SetText(string(b))
	row.Square.SetDirty(false)
	return row, nil
}
