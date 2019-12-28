package internalcmds

import (
	"fmt"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/ui"
)

func GoRename(args0 *core.InternalCmdArgs) error {
	erow := args0.ERow
	part := args0.Part

	if !erow.Info.IsFileButNotDir() {
		return fmt.Errorf("not a file")
	}

	if erow.Row.HasState(ui.RowStateEdited) {
		return fmt.Errorf("row has edits, save first")
	}

	// new name argument "to"
	args := part.Args[1:]
	if len(args) < 1 {
		return fmt.Errorf("expecting at least 1 argument")
	}
	to := args[len(args)-1].UnquotedStr()

	// allow other args
	otherArgs := []string{}
	for i := 0; i < len(args)-1; i++ {
		otherArgs = append(otherArgs, args[i].UnquotedStr())
	}

	// id offset to rename "from"
	offset := erow.Row.TextArea.TextCursor.Index()

	// command
	offsetStr := fmt.Sprintf("%v:#%v", erow.Info.Name(), offset)
	cargs := []string{"gorename", "-offset", offsetStr, "-to", to}
	cargs = append(cargs, otherArgs...)

	reloadOnNoErr := func(err error) {
		if err == nil {
			erow.Ed.UI.RunOnUIGoRoutine(func() {
				erow.Reload()
			})
		}
	}

	core.ExternalCmdFromArgs(erow, cargs, reloadOnNoErr)

	return nil
}
