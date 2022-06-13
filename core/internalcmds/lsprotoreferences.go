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

func LSProtoReferences(args0 *core.InternalCmdArgs) error {
	ed := args0.Ed
	erow := args0.ERow

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
		return printLocations(rw, locs, erow2.Info.Dir())
	})

	return nil
}

func printLocations(rw io.ReadWriter, locations []*lsproto.Location, baseDir string) error {
	fmt.Fprintf(rw, "lsproto references:")
	if len(locations) == 0 {
		fmt.Fprintf(rw, " no results\n")
		return nil
	}
	fmt.Fprintf(rw, "\n")
	for _, loc := range locations {
		filename, err := lsproto.UrlToAbsFilename(string(loc.Uri))
		if err != nil {
			return err
		}

		// use basedir to output filename
		if baseDir != "" {
			if u, err := filepath.Rel(baseDir, filename); err == nil {
				filename = u
			}
		}

		line, col := loc.Range.Start.OneBased()
		fmt.Fprintf(rw, "\t%v:%v:%v\n", filename, line, col)
	}
	return nil
}
