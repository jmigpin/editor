package internalcmds

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/core/lsproto"
	"github.com/jmigpin/editor/util/iout/iorw"
)

func LSProtoCallHierarchyIncomingCalls(args *core.InternalCmdArgs) error {
	return lsprotoCallHierarchyCalls(args, lsproto.IncomingChct)
}
func LSProtoCallHierarchyOutgoingCalls(args *core.InternalCmdArgs) error {
	return lsprotoCallHierarchyCalls(args, lsproto.OutgoingChct)
}

func lsprotoCallHierarchyCalls(args *core.InternalCmdArgs, typ lsproto.CallHierarchyCallType) error {
	ed := args.Ed

	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}

	if !erow.Info.IsFileButNotDir() {
		return fmt.Errorf("not a file")
	}

	// create new erow to run on
	dir := filepath.Dir(erow.Info.Name())
	info := erow.Ed.ReadERowInfo(dir)
	erow2 := core.NewBasicERow(info, erow.Row.PosBelow())
	iorw.Append(erow2.Row.Toolbar.RW(), []byte(" | Stop"))
	erow2.Flash()

	// NOTE: args0.Ctx will end at func exit

	erow2.Exec.RunAsync(func(ctx context.Context, rw io.ReadWriter) error {
		// NOTE: not running in UI goroutine here

		ta := erow.Row.TextArea
		mcalls, err := ed.LSProtoMan.CallHierarchyCalls(ctx, erow.Info.Name(), ta.RW(), ta.CursorIndex(), typ)
		if err != nil {
			return err
		}
		str, err := lsproto.ManagerCallHierarchyCallsToString(mcalls, typ, erow2.Info.Dir())
		if err != nil {
			return err
		}
		fmt.Fprintf(rw, str)
		return nil
	})

	return nil
}
