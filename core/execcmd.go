package core

import (
	"context"
	"io"

	"github.com/jmigpin/editor/util/osutil"
)

func ExecCmd(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := osutil.NewCmd(ctx, args...)
	cmd.Dir = dir
	return osutil.RunCmdStdoutAndStderrInErr(cmd)
}

func ExecCmdStdin(ctx context.Context, dir string, in io.Reader, args ...string) ([]byte, error) {
	cmd := osutil.NewCmd(ctx, args...)
	cmd.Dir = dir
	if err := cmd.SetupStdInOutErr(in, nil, nil); err != nil {
		return nil, err
	}
	return osutil.RunCmdStdoutAndStderrInErr(cmd)
}
