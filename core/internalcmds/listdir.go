package internalcmds

import (
	"flag"
	"fmt"
	"io"

	"github.com/jmigpin/editor/core"
)

func ListDir(args *core.InternalCmdArgs) error {
	// setup flagset
	fs := flag.NewFlagSet("ListDir", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // don't output to stderr
	subFlag := fs.Bool("sub", false, "list subdirectories/files")
	hiddenFlag := fs.Bool("hidden", false, "list hidden files")
	if err := parseFlagSetHandleUsage(args, fs); err != nil {
		return err
	}

	//----------

	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}

	if !erow.Info.IsDir() {
		return fmt.Errorf("not a directory")
	}

	core.ListDirERow(erow, erow.Info.Name(), *subFlag, *hiddenFlag)

	return nil
}
