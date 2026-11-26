package internalcmds

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/core/toolbarparser"
)

func Run(args *core.InternalCmdArgs) error {
	opts, err := parseRunOptions(args.Part)
	if err != nil {
		return err
	}

	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}
	if opts.script != "" {
		core.ExternalCmd(erow, []string{opts.script}, nil, nil, core.ExternalCmdModeShellScript)
		return nil
	}
	if opts.shellArgsMode {
		core.ExternalCmd(erow, opts.shellArgs, nil, nil, core.ExternalCmdModeShellArgs)
		return nil
	}

	ta := erow.Row.TextArea
	a, b, ok := ta.Cursor().SelectionIndexes()
	if !ok || a == b {
		return fmt.Errorf("no text selected in row textarea")
	}
	selected, err := ta.EditCtx().RW.ReadFastAt(a, b-a)
	if err != nil {
		return err
	}
	value := strings.TrimSpace(string(selected))
	if value == "" {
		return fmt.Errorf("no text selected in row textarea")
	}
	return erow.RunSelectedText(value)
}

//----------

type runOptions struct {
	textareaSelection bool
	script            string
	shellArgsMode     bool
	shellArgs         []string
}

func parseRunOptions(part *toolbarparser.Part) (*runOptions, error) {
	fs := flag.NewFlagSet("Run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	opts := &runOptions{}
	fs.BoolVar(&opts.textareaSelection, "textareaSelection", false, "run the selected text in the active row textarea")
	fs.StringVar(&opts.script, "script", "", "run a shell script in the active row (or a new row for files)")
	fs.BoolVar(&opts.shellArgsMode, "shellArgs", false, "run a shell command with separate arguments in the active row (or a new row for files)")
	if err := parseFlagSetHandleUsageArgs(fs, part.ArgsUnquoted()[1:]); err != nil {
		return nil, err
	}
	modes := 0
	for _, on := range []bool{opts.textareaSelection, opts.script != "", opts.shellArgsMode} {
		if on {
			modes++
		}
	}
	if modes != 1 || (opts.shellArgsMode && fs.NArg() == 0) || (!opts.shellArgsMode && fs.NArg() != 0) {
		return nil, fmt.Errorf("usage: Run -textareaSelection | Run -script <script> | Run -shellArgs -- <command> [args...] (see Run -h)")
	}
	opts.shellArgs = fs.Args()
	return opts, nil
}
