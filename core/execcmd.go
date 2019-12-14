package core

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/jmigpin/editor/util/osutil"
)

func ExecCmd(ctx context.Context, dir string, args ...string) ([]byte, error) {
	return ExecCmdStdin(ctx, dir, nil, args...)
}

func ExecCmdStdin(ctx context.Context, dir string, in io.Reader, args ...string) ([]byte, error) {
	ecmd := osutil.ExecCmdCtxWithAttr(ctx, args[0], args[1:]...)
	ecmd.Dir = dir
	ecmd.Stdin = in
	sout, serr, err := osutil.ExecCmdRunOutputs(ecmd)
	if err != nil {
		return nil, fmt.Errorf("%v: err=(%v), stderr=(%v), stdout=(%v)", args[0], err, string(serr), string(sout))
	}
	return sout, nil
}

func HasExecErrNotFound(err error) bool {
	return strings.Index(err.Error(), exec.ErrNotFound.Error()) >= 0
}
