package internalcmds

import (
	"fmt"
	"strconv"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/util/parseutil"
)

func GotoLine(args0 *core.InternalCmdArgs) error {
	erow := args0.ERow
	part := args0.Part

	args := part.Args[1:]
	if len(args) != 1 {
		return fmt.Errorf("expecting 1 argument")
	}

	line0, err := strconv.ParseUint(args[0].Str(), 10, 64)
	if err != nil {
		return err
	}
	line := int(line0)

	ta := erow.Row.TextArea
	index, err := parseutil.LineColumnIndex(ta.RW(), line, 0)
	if err != nil {
		return err
	}

	// goto index
	ta.Cursor().SetIndexSelectionOff(index)

	erow.MakeIndexVisibleAndFlash(index)

	return nil
}
