package internalcmds

import (
	"fmt"

	"github.com/jmigpin/editor/core"
)

func RuneCodes(args *core.InternalCmdArgs) error {
	erow := args.ERow

	ta := erow.Row.TextArea
	tc := ta.TextCursor
	if !tc.SelectionOn() {
		return fmt.Errorf("no text selected")
	}
	b, err := tc.Selection()
	if err != nil {
		return err
	}

	s := "runecodes:\n"
	for i, ru := range string(b) {
		s += fmt.Sprintf("\t%v: %c, %v\n", i, ru, int(ru))
	}
	erow.Ed.Messagef(s)

	return nil
}
