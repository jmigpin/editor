package internalcmds

import (
	"bytes"
	"errors"
	"flag"
	"fmt"

	"github.com/jmigpin/editor/core"
)

func ListDir(args *core.InternalCmdArgs) error {
	erow, err := args.ERowOrErr()
	if err != nil {
		return err
	}

	baseDir := erow.Info.Dir()
	parsed, err := core.ParseListDirCmdArgs(args.Part.ArgsUnquoted()[1:], core.ListDirCmdConfig{
		BaseDir:    baseDir,
		DecodePath: args.Ed.HomeVars.Decode,
		EncodePath: args.Ed.HomeVars.EncodeShortest,
	})
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			buf := &bytes.Buffer{}
			core.ListDirFlagSetUsage(buf)
			return fmt.Errorf("%w\n%v", err, buf.String())
		}
		return err
	}

	if len(parsed.Sources) == 1 && parsed.Sources[0].AddedFilepath == "" && !erow.Info.IsDir() {
		return fmt.Errorf("not a directory")
	}
	core.ListDirERowOptionsSources(erow, parsed.Sources, parsed.Opts)

	return nil
}
