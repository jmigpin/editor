package internalcmds

import (
	"bytes"
	"flag"
	"fmt"
	"io"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/util/osutil"
)

func Edit(args *core.InternalCmdArgs) error {
	fs := flag.NewFlagSet("Edit", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pipeFlag := fs.Bool("pipe", false, "replace selection or buffer with stdout from command")
	if err := parseFlagSetHandleUsage(args, fs); err != nil {
		return err
	}

	if !*pipeFlag {
		return fmt.Errorf("unsupported edit mode: use Edit -pipe <cmd...>")
	}
	return editPipe(args, fs.Args())
}

//----------

func editPipe(args *core.InternalCmdArgs, cmdArgs []string) error {
	if len(cmdArgs) == 0 {
		return fmt.Errorf("missing command: use Edit -pipe <cmd...>")
	}

	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}

	ta := erow.Row.TextArea
	ectx := ta.EditCtx()
	a, b, ok := ectx.C.SelectionIndexes()
	if !ok {
		a = ectx.RW.Min()
		b = ectx.RW.Max()
	}

	src, err := ectx.RW.ReadFastAt(a, b-a)
	if err != nil {
		return err
	}

	dir := erow.Info.Dir()
	out, err := osutil.RunCmdStdin(args.Ctx, dir, bytes.NewReader(src), cmdArgs...)
	if err != nil {
		return err
	}

	ta.BeginUndoGroup()
	defer ta.EndUndoGroup()
	if err := ectx.RW.OverwriteAt(a, b-a, out); err != nil {
		return err
	}
	//ectx.C.SetSelection(a, a+len(out))
	//erow.MakeRangeVisibleAndFlash(a, len(out))

	return nil
}
