/*
Build with:
$ go build -buildmode=plugin gotodefinition.go
*/

package main

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"time"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/core/parseutil"
)

func OnLoad(ed *core.Editor) {
	// default contentcmds at: github.com/jmigpin/editor/core/contentcmds/init.go
	core.ContentCmds.Remove("gotodefinition") // remove default
	core.ContentCmds.Prepend("my_gotodefinition", goToDefinition)
}

func goToDefinition(erow *core.ERow, index int) (handled bool, err error) {
	if erow.Info.IsDir() {
		return false, nil
	}
	if path.Ext(erow.Info.Name()) != ".go" {
		return false, nil
	}

	// timeout for the cmd to run
	timeout := 8000 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// it's a go file, return true from here

	// go guru args
	//position := fmt.Sprintf("%v:#%v", erow.Info.Name(), index)
	//args := []string{"guru", "definition", position}

	// godef args
	args := []string{"godef", "-f", erow.Info.Name(), "-o", fmt.Sprintf("%v", index)}

	// execute external cmd
	dir := filepath.Dir(erow.Info.Name())
	out, err := core.ExecCmd(ctx, dir, args...)
	if err != nil {
		return true, err
	}

	// parse external cmd output
	filePos, err := parseutil.ParseFilePos(string(out))
	if err != nil {
		return true, err
	}

	// place under the calling row
	rowPos := erow.Row.PosBelow()

	conf := &core.OpenFileERowConfig{
		FilePos:               filePos,
		RowPos:                rowPos,
		FlashVisibleOffsets:   true,
		NewIfNotExistent:      true,
		NewIfOffsetNotVisible: true,
	}
	core.OpenFileERow(erow.Ed, conf)

	return true, nil
}
