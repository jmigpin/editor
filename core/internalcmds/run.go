package internalcmds

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/jmigpin/editor/core"
)

func Run(args *core.InternalCmdArgs) error {
	fs := flag.NewFlagSet("Run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	textareaSelection := fs.Bool("textareaSelection", false, "run the selected text in the active row textarea")
	if err := parseFlagSetHandleUsage(args, fs); err != nil {
		return err
	}
	if !*textareaSelection || fs.NArg() != 0 {
		return fmt.Errorf("usage: Run -textareaSelection (see Run -h)")
	}

	erow, err := args.ERowOrErr()
	if err != nil {
		return err
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
