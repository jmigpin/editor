package internalcmds

import (
	"fmt"

	"github.com/jmigpin/editor/v2/core"
)

func RuneCodes(args *core.InternalCmdArgs) error {
	erow := args.ERow

	ta := erow.Row.TextArea
	b, ok := ta.EditCtx().Selection()
	if !ok {
		return fmt.Errorf("no text selected")
	}

	s := "runecodes:\n"
	for i, ru := range string(b) {
		s += fmt.Sprintf("\t%v: %c, %v\n", i, ru, int(ru))
	}
	erow.Ed.Messagef(s)

	return nil
}
