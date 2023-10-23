/*
Build with:
$ go build -buildmode=plugin gotodefinition_godef.go
*/

package main

import (
	"bytes"
	"context"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil/reslocparser"
)

func OnLoad(ed *core.Editor) {
	// default contentcmds at: github.com/jmigpin/editor/core/contentcmds/init.go
	core.ContentCmds.Remove("gotodefinition") // remove default
	core.ContentCmds.Prepend("gotodefinition_godef", goToDefinition)
}

func goToDefinition(ctx0 context.Context, erow *core.ERow, index int) (err error, handled bool) {
	if erow.Info.IsDir() {
		return nil, false
	}
	if path.Ext(erow.Info.Name()) != ".go" {
		return nil, false
	}

	// timeout for the cmd to run
	timeout := 8000 * time.Millisecond
	ctx, cancel := context.WithTimeout(ctx0, timeout)
	defer cancel()

	// it's a go file, return true from here

	// godef args
	args := []string{osutil.ExecName("godef"), "-i", "-f", erow.Info.Name(), "-o", strconv.Itoa(index)}

	// godef can read from stdin: use textarea bytes
	bin, err := erow.Row.TextArea.Bytes()
	if err != nil {
		return err, true
	}
	in := bytes.NewBuffer(bin)

	// execute external cmd
	dir := filepath.Dir(erow.Info.Name())
	out, err := osutil.RunCmdStdin(ctx, dir, in, args...)
	if err != nil {
		return err, true
	}

	// parse external cmd output
	filePos, err := reslocparser.ParseFilePos(out, 0)
	if err != nil {
		return err, true
	}

	erow.Ed.UI.RunOnUIGoRoutine(func() {
		// place under the calling row
		rowPos := erow.Row.PosBelow() // needs ui goroutine

		conf := &core.OpenFileERowConfig{
			FilePos:               filePos,
			RowPos:                rowPos,
			FlashVisibleOffsets:   true,
			NewIfNotExistent:      true,
			NewIfOffsetNotVisible: true,
		}
		core.OpenFileERow(erow.Ed, conf) // needs ui goroutine
	})

	return nil, true
}
