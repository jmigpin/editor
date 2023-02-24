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

	if erow.Row.HasState(ui.RowStateEdited | ui.RowStateFsDiffer) {
		return fmt.Errorf("row has edits, save first")
	}

	args := part.Args[1:]
	if len(args) < 1 {
		return fmt.Errorf("expecting at least 1 argument")
	}

	// optional "-all" (only as first arg) for full rename (not an option on either gorename/gopls)
	isF := false
	if args[0].String() == "-all" {
		isF = true
		args = args[1:]
	}

	// new name argument "to"
	to := args[len(args)-1].UnquotedString()

	// allow other args
	otherArgs := []string{}
	for i := 0; i < len(args)-1; i++ {
		otherArgs = append(otherArgs, args[i].UnquotedString())
	}

	// id offset to rename "from"
	offset := erow.Row.TextArea.CursorIndex()

	// command
	offsetStr := fmt.Sprintf("%v:#%v", erow.Info.Name(), offset)
	cargs := []string{}
	if isF {
		cargs = []string{"gorename", "-offset", offsetStr, "-to", to}
		cargs = append(cargs, otherArgs...)
	} else {
		cargs = append([]string{"gopls", "rename"}, append(otherArgs, []string{"-w", offsetStr, to}...)...)
	}

	// TODO: reload all changed files (check stdout)

	reloadOnNoErr := func(err error) {
		if err == nil {
			erow.Ed.UI.RunOnUIGoRoutine(func() {
				if err := erow.Reload(); err != nil {
					erow.Ed.Error(err)
				}
			})
		}
	}

	core.ExternalCmdFromArgs(erow, cargs, reloadOnNoErr, nil)

	return nil
}
