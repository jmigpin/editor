package osutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func HomeEnvVar() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

//----------

func FilepathHasDirPrefix(s, prefix string) bool {
	// ensure it ends in separator
	sep := string(filepath.Separator)
	if !strings.HasSuffix(prefix, sep) {
		prefix += sep
	}

	return strings.HasPrefix(s, prefix)
}

// Result does not start with separator.
func FilepathSplitAt(s string, n int) string {
	if n > len(s) {
		return ""
	}
	for ; n < len(s); n++ {
		if s[n] != filepath.Separator {
			break
		}
	}
	return s[n:]
}

func FilepathClean(s string) string {
	return filepath.Clean(s)
}

//----------

func RunExecCmdCtxWithAttrAndGetOutputs(ctx context.Context, dir string, in io.Reader, args []string, env []string) ([]byte, error) {
	ecmd := ExecCmdCtxWithAttr(ctx, args)
	ecmd.Dir = dir
	ecmd.Stdin = in
	ecmd.Env = env
	return RunExecCmdAndGetStdout(ecmd)
}

func ExecCmdCtxWithAttr(ctx context.Context, args []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	SetupExecCmdSysProcAttr(cmd)
	return cmd
}

//----------

// Adds stderr to err if it happens.
func RunExecCmdAndGetStdout(ecmd *exec.Cmd) ([]byte, error) {
	bout, berr, err := RunExecCmdAndGetOutputs(ecmd)
	if err != nil {
		serr := strings.TrimSpace(string(berr))
		if serr != "" {
			serr = fmt.Sprintf(", stderr(%v)", serr)
		}
		sout := strings.TrimSpace(string(bout))
		if sout != "" {
			sout = fmt.Sprintf(", stdout(%v)", sout)
		}
		return nil, fmt.Errorf("%v: %v%v%v", ecmd.Path, err, serr, sout)
	}
	return bout, nil
}

func RunExecCmdAndGetOutputs(ecmd *exec.Cmd) (sout []byte, serr []byte, _ error) {
	if ecmd.Stdout != nil {
		return nil, nil, fmt.Errorf("stdout already set")
	}
	if ecmd.Stderr != nil {
		return nil, nil, fmt.Errorf("stderr already set")
	}

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	ecmd.Stdout = &stdoutBuf
	ecmd.Stderr = &stderrBuf

	err := ecmd.Run()
	return stdoutBuf.Bytes(), stderrBuf.Bytes(), err
}
