package internalcmds

import (
	"fmt"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/util/parseutil"
	"github.com/jmigpin/editor/util/uiutil/event"
)

func CopyPosition(args *core.InternalCmdArgs) error {
	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}

	var s string
	switch {
	case erow.Info.IsFileButNotDir():
		ta := erow.Row.TextArea
		ci := ta.CursorIndex()
		rd := ta.RW()
		line, col, err := parseutil.IndexLineColumn(rd, ci)
		if err != nil {
			return err
		}
		s = fmt.Sprintf("%v:%v:%v", erow.Info.Name(), line, col)
	case erow.Info.IsDir():
		s = erow.Info.Name()
	default:
		return fmt.Errorf("not a file or dir")
	}

	erow.Ed.UI.SetClipboardData(event.CIClipboard, s)
	erow.Ed.Message("copyposition:\n\t" + s)

	return nil
}
