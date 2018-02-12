package cmdutil

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/jmigpin/editor/core/toolbardata"
	"github.com/jmigpin/editor/ui"
)

func GoRename(erow ERower, part *toolbardata.Part) {
	if !erow.IsRegular() {
		erow.Ed().Errorf("not a regular file")
		return
	}

	if erow.Row().HasState(ui.EditedRowState) {
		erow.Ed().Errorf("row has edits, save first")
		return
	}

	// new name argument "to"
	a := part.Args[1:]
	if len(a) != 1 {
		erow.Ed().Errorf("GoRename: expecting 1 argument")
		return
	}
	to := a[0].UnquotedStr()

	// filename and id offset to rename "from"
	offset := erow.Row().TextArea.CursorIndex()
	offsetStr := fmt.Sprintf("%v:#%v", erow.Filename(), offset)

	// run on a goroutine to prevent hanging the UI (could be slow)
	go func() {
		str, err := runGoRename(offsetStr, to)
		if str != "" {
			erow.Ed().Messagef(str)
		}
		if err != nil {
			erow.Ed().Error(err)
		} else {
			erow.Ed().UI().RunOnUIGoRoutine(func() {
				ReloadRow(erow)
			})
		}
	}()
}
func runGoRename(offsetStr, to string) (string, error) {
	ctx := context.Background()
	c := exec.CommandContext(ctx, "gorename", "-v", "-offset", offsetStr, "-to", to)

	// pipe string to command stdin
	//c.Stdin = strings.NewReader(str)

	// output
	var ob, eb bytes.Buffer
	c.Stdout = &ob
	c.Stderr = &eb

	err := c.Run()
	if err != nil {
		// ignore err, get error string from stderr
		err2 := fmt.Errorf("%v", eb.String())
		return "", err2
	}

	return ob.String(), nil
}
