package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"unicode"
)

func ExecCmd(ctx context.Context, dir string, args ...string) ([]byte, error) {
	return ExecCmdStdin(ctx, dir, nil, args...)
}

func ExecCmdStdin(ctx context.Context, dir string, in io.Reader, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Stdin = in
	b, err := cmd.CombinedOutput()
	if err != nil {
		w := bytes.TrimRightFunc(b, unicode.IsSpace)
		return nil, fmt.Errorf("%v: %v: %v", args[0], err, string(w))
	}
	return b, nil
}

func HasExecErrNotFound(err error) bool {
	return strings.Index(err.Error(), exec.ErrNotFound.Error()) >= 0
}
