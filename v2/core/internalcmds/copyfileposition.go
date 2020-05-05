package internalcmds

import (
	"fmt"

	"github.com/jmigpin/editor/v2/core"
	"github.com/jmigpin/editor/v2/util/parseutil"
)

func CopyFilePosition(args0 *core.InternalCmdArgs) error {
	erow := args0.ERow

	if !erow.Info.IsFileButNotDir() {
		return fmt.Errorf("not a file")
	}

	ta := erow.Row.TextArea
	ci := ta.CursorIndex()
	rd := ta.RW()
	line, col, err := parseutil.IndexLineColumn(rd, ci)
	if err != nil {
		return err
	}

	s := fmt.Sprintf("copyfileposition:\n\t%v:%v:%v", erow.Info.Name(), line, col)
	erow.Ed.Messagef(s)

	return nil
}
