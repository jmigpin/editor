package internalcmds

import (
	"bytes"
	"flag"
	"fmt"

	"github.com/jmigpin/editor/core"
)

func parseFlagSetHandleUsage(args *core.InternalCmdArgs, fs *flag.FlagSet) error {
	err := fs.Parse(args.Part.ArgsStrings()[1:])

	// improve error with usage
	if err == flag.ErrHelp {
		buf := &bytes.Buffer{}
		fs.SetOutput(buf)
		fs.Usage()
		err = fmt.Errorf("%w\n%v", err, buf.String())
	}

	return err
}
