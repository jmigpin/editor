package internalcmds

import (
	"fmt"
	"strconv"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/util/parseutil"
)

func GotoLine(args *core.InternalCmdArgs) error {
	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}

	args2 := args.Part.Args[1:]
	if len(args2) != 1 {
		return fmt.Errorf("expecting 1 argument")
	}

	line0, err := strconv.ParseUint(args2[0].String(), 10, 64)
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
