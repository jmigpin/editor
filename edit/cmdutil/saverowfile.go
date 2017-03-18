package cmdutil

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/jmigpin/editor/ui"
)

func SaveRowsFiles(ed Editorer) {
	for _, c := range ed.UI().Layout.Cols.Cols {
		for _, r := range c.Rows {
			saveRowFile2(ed, r, true)
		}
	}
}
func SaveRowFile(ed Editorer, row *ui.Row) {
	saveRowFile2(ed, row, false)
}
func saveRowFile2(ed Editorer, row *ui.Row, tolerant bool) {
	tsd := ed.RowToolbarStringData(row)
	// file might not exist yet, so getting from filepath
	filename := tsd.FirstPartFilepath()

	// best effort to disable/enable file watcher, ignore errors
	_ = ed.FilesWatcherRemove(filename)
	defer func() {
		_ = ed.FilesWatcherAdd(filename)
	}()

	// save
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		ed.Error(err)
		return
	}
	defer f.Close()
	data := []byte(row.TextArea.Str())
	_, err = f.Write(data)
	if err != nil {
		ed.Error(err)
		return
	}
	row.Square.SetDirty(false)
	row.Square.SetCold(false)

	// run go imports for go files
	if path.Ext(filename) == ".go" {
		err := runGoImports(filename)
		if err != nil {
			// ignore errors, can catch them when compiling
		} else {
			ReloadRow(ed, row)
		}
	}
}
func runGoImports(filename string) error {
	ctx := context.Background()
	c := exec.CommandContext(ctx, "goimports", "-w", filename)

	// combined output
	var b bytes.Buffer
	c.Stdout = &b
	c.Stderr = &b

	err := c.Run()
	if err != nil {
		err2 := fmt.Errorf("%v\n%v", err, b.String())
		return err2
	}
	return nil
}
