package internalcmds

import (
	"flag"
	"fmt"
	"strings"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/util/parseutil"
	"github.com/jmigpin/editor/util/uiutil/event"
)

func CopyPosition(args *core.InternalCmdArgs) error {
	opt := copyPositionOpts{clipboard: "both"}
	fs := flag.NewFlagSet("CopyPosition", flag.ContinueOnError)
	fs.BoolVar(&opt.quiet, "quiet", true, "copy without reporting to the messages row")
	fs.StringVar(&opt.clipboard, "clipboard", opt.clipboard, "clipboard target: clipboard, primary, or both")
	if err := parseFlagSetHandleUsage(args, fs); err != nil {
		return err
	}
	target, err := opt.clipboardTarget()
	if err != nil {
		return err
	}

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

	target.set(erow, s)
	if !opt.quiet {
		erow.Ed.Message("copyposition:\n\t" + s)
	}

	return nil
}

//----------

type copyPositionOpts struct {
	quiet     bool
	clipboard string
}

func (opt copyPositionOpts) clipboardTarget() (copyPositionClipboardTarget, error) {
	switch strings.ToLower(opt.clipboard) {
	case "clipboard":
		return copyPositionClipboardTarget{clipboard: true}, nil
	case "primary":
		return copyPositionClipboardTarget{primary: true}, nil
	case "both":
		return copyPositionClipboardTarget{primary: true, clipboard: true}, nil
	default:
		return copyPositionClipboardTarget{}, fmt.Errorf("copyposition: invalid clipboard target %q", opt.clipboard)
	}
}

type copyPositionClipboardTarget struct {
	primary   bool
	clipboard bool
}

func (target copyPositionClipboardTarget) set(erow *core.ERow, s string) {
	if target.primary {
		erow.Ed.UI.SetClipboardData(event.CIPrimary, s)
	}
	if target.clipboard {
		erow.Ed.UI.SetClipboardData(event.CIClipboard, s)
	}
}
