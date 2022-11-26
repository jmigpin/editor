package internalcmds

import (
	"fmt"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/util/iout/iorw/rwedit"
)

func Replace(args0 *core.InternalCmdArgs) error {
	erow := args0.ERow
	part := args0.Part

	args := part.Args[1:]
	if len(args) != 2 {
		return fmt.Errorf("expecting 2 arguments")
	}

	old, new := args[0].UnquotedString(), args[1].UnquotedString()

	ta := erow.Row.TextArea
	ta.BeginUndoGroup()
	defer ta.EndUndoGroup()
	replaced, err := rwedit.Replace(ta.EditCtx(), old, new)
	if err != nil {
		return err
	}
	if !replaced {
		return fmt.Errorf("string not replaced: %q", old)
	}
	return nil
}
