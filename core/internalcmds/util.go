package internalcmds

import (
	"bytes"
	"flag"

	"github.com/jmigpin/editor/core"
)

func parseFlagSetHandleUsage(args *core.InternalCmdArgs, fs *flag.FlagSet) error {
	return parseFlagSetHandleUsageArgs(fs, args.Part.ArgsStrings()[1:])
}

func parseFlagSetHandleUsageArgs(fs *flag.FlagSet, args []string) error {
	err := fs.Parse(args)

	// improve error with usage
	if err == flag.ErrHelp {
		buf := &bytes.Buffer{}
		fs.SetOutput(buf)
		fs.Usage()
		err = core.NewHelpRequestError(fs.Name(), buf.String())
	}

	return err
}
