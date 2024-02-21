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

func LSProtoReferences(args *core.InternalCmdArgs) error {
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
		locs, err := ed.LSProtoMan.TextDocumentReferences(ctx, erow.Info.Name(), ta.RW(), ta.CursorIndex())
		if err != nil {
			return err
		}

		// print locations
		str, err := lsproto.LocationsToString(locs, erow2.Info.Dir())
		if err != nil {
			return err
		}
		fmt.Fprintf(rw, "lsproto references:")
		if len(locs) == 0 {
			fmt.Fprintf(rw, " no results\n")
			return nil
		}
		fmt.Fprintf(rw, "\n%v", str)
		return nil
	})

	return nil
}
