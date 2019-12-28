package internalcmds

import (
	"fmt"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/util/uiutil/widget/textutil"
)

func Replace(args0 *core.InternalCmdArgs) error {
	erow := args0.ERow
	part := args0.Part

	args := part.Args[1:]
	if len(args) != 2 {
		return fmt.Errorf("expecting 2 arguments")
	}

	old, new := args[0].UnquotedStr(), args[1].UnquotedStr()

	replaced, err := textutil.Replace(erow.Row.TextArea.TextEdit, old, new)
	if err != nil {
		return err
	}
	if !replaced {
		return fmt.Errorf("string not replaced: %q", old)
	}
	return nil
}
