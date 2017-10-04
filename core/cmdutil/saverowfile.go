package cmdutil

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path"
	"strings"
)

func SaveRowsFiles(ed Editorer) {
	for _, erow := range ed.ERows() {
		SaveRowFile(erow)
	}
}
func SaveRowFile(erow ERower) {
	row := erow.Row()
	str := row.TextArea.Str()

	// run go imports for go content, updates content string
	fp := erow.Filename()
	if path.Ext(fp) == ".go" {
		u, err := runGoImports(str)
		if err != nil {
			// ignore errors, can catch them when compiling
		} else {
			// Setting str to a full new string from goimports will
			// have the cursor be set at the end of the string.
			// This becomes annoying when doing undo/redo.
			// The altered history is to keep the previous cursor
			// position and make it pleasant.

			ci := row.TextArea.CursorIndex()

			str = u
			row.TextArea.SetStrClear(str, false, false)

			row.TextArea.History().MergeLastTwoEditsAndAddPosition(ci)
		}
	}

	err := erow.SaveContent(str)
	if err != nil {
		erow.Ed().Error(err)
	}
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
