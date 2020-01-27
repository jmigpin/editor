package core

import (
	"context"
	"io"

	"github.com/jmigpin/editor/util/osutil"
)

func ExecCmd(ctx context.Context, dir string, args ...string) ([]byte, error) {
	return ExecCmdStdin(ctx, dir, nil, args...)
}

func ExecCmdStdin(ctx context.Context, dir string, in io.Reader, args ...string) ([]byte, error) {
	return osutil.RunExecCmdCtxWithAttrAndGetOutputs(ctx, dir, in, args...)
}
