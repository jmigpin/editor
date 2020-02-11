package core

import (
	"context"
	"io"

	"github.com/jmigpin/editor/util/osutil"
)

func ExecCmd(ctx context.Context, dir string, args ...string) ([]byte, error) {
	return ExecCmdStdin(ctx, dir, nil, args...)
}

func ExecCmdStdin(ctx context.Context, dir string, rd io.Reader, args ...string) ([]byte, error) {
	cmd := osutil.NewCmd(ctx, args...)
	cmd.Dir = dir
	return osutil.RunCmdStdoutAndStderrInErr(cmd, rd)
}
