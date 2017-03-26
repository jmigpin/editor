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
	content := row.TextArea.Str()

	// run go imports for go files, updates content string
	fp, fi, ok := erow.FileInfo()
	if ok && !fi.IsDir() && path.Ext(fp) == ".go" {
		u, err := runGoImports(content)
		if err != nil {
			// ignore errors, can catch them when compiling
		} else {
			content = u
			row.TextArea.SetStrClear(content, false, false)
		}
	}

	err := erow.SaveContent(content)
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
