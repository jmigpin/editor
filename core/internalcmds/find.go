package internalcmds

import (
	"fmt"
	"strings"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/util/iout/iorw/rwedit"
)

func Find(args0 *core.InternalCmdArgs) error {
	erow := args0.ERow
	part := args0.Part

	args := part.Args[1:]
	if len(args) < 1 {
		return fmt.Errorf("expecting argument")
	}
	var str string
	if len(args) == 1 {
		str = args[0].UnquotedStr()
	} else {
		// join args
		a, b := args[0].Pos, args[len(args)-1].End
		s := part.Data.Str[a:b]
		str = strings.TrimSpace(s)
	}

	found, err := rwedit.Find(args0.Ctx, erow.Row.TextArea.EditCtx(), str)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("string not found: %q", str)
	}

	// flash
	ta := erow.Row.TextArea
	if a, b, ok := ta.Cursor().SelectionIndexes(); ok {
		erow.MakeRangeVisibleAndFlash(a, b-a)
	}

	return nil
}
