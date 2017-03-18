package cmdutil

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

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

	content := row.TextArea.Str()

	// run go imports for go files, updates content string
	if path.Ext(filename) == ".go" {
		u, err := runGoImports(content)
		if err != nil {
			// ignore errors, can catch them when compiling
		} else {
			content = u
			row.TextArea.SetStrClear(content, false, false)
		}
	}

	// save
	flags := os.O_WRONLY | os.O_TRUNC | os.O_CREATE
	f, err := os.OpenFile(filename, flags, 0644)
	if err != nil {
		ed.Error(err)
		return
	}
	defer f.Close()
	data := []byte(content)
	_, err = f.Write(data)
	if err != nil {
		ed.Error(err)
		return
	}

	row.Square.SetDirty(false)
	row.Square.SetCold(false)
}
func runGoImports(str string) (string, error) {
	ctx := context.Background()
	c := exec.CommandContext(ctx, "goimports")

	// pipe string to command stdin
	c.Stdin = strings.NewReader(str)

	// output
	var ob, eb bytes.Buffer
	c.Stdout = &ob
	c.Stderr = &eb

	err := c.Run()
	if err != nil {
		// ignore err, get error string from stdout
		err2 := fmt.Errorf("%v", eb.String())
		return "", err2
	}

	return ob.String(), nil
}
