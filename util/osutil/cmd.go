package osutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

type Cmd struct {
	*exec.Cmd

	PreOutputCallback func()

	ctx       context.Context
	CancelCtx context.CancelFunc

	NoEnsureStop bool
	ensureStop   struct {
		sync.Mutex
		on bool
	}
}

// If Start() is not called, CancelCtx() must be called to clear resources.
func NewCmd(ctx context.Context, args ...string) *Cmd {
	ctx2, cancel := context.WithCancel(ctx)
	c := exec.CommandContext(ctx2, args[0], args[1:]...) // panic on empty args
	cmd := &Cmd{Cmd: c, ctx: ctx2, CancelCtx: cancel}
	return cmd
}

//----------

// If Start() returns no error, Wait() must be called to clear resources.
func (cmd *Cmd) Start() error {
	if err := cmd.start2(); err != nil {
		cmd.CancelCtx()
		return err
	}
	return nil
}

func (cmd *Cmd) start2() error {
	if cmd.Cmd.SysProcAttr == nil {
		SetupExecCmdSysProcAttr(cmd.Cmd)
	}

	if err := cmd.Cmd.Start(); err != nil {
		return err
	}

	// TODO: ensure it is called before the first stdout/stderr write (works since the process takes longer to launch and write back, but in theory this could be called after some output comes out)
	if cmd.PreOutputCallback != nil {
		cmd.PreOutputCallback()
	}

	go func() {
		select {
		case <-cmd.ctx.Done():
			cmd.ensureStopNow()
		}
	}()

	return nil
}

func (cmd *Cmd) Wait() error {
	// Explanations on possible hangs.
	// https://github.com/golang/go/issues/18874#issuecomment-277280139

	defer func() {
		cmd.enableEnsureStop(false) // no need to kill process anymore
		cmd.CancelCtx()
	}()
	return cmd.Cmd.Wait()
}

func (cmd *Cmd) Run() error {
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Wait()
}

//----------

func (cmd *Cmd) enableEnsureStop(on bool) {
	cmd.ensureStop.Lock()
	defer cmd.ensureStop.Unlock()
	cmd.ensureStop.on = on
}

func (cmd *Cmd) ensureStopNow() {
	cmd.ensureStop.Lock()
	defer cmd.ensureStop.Unlock()
	if cmd.ensureStop.on {
		cmd.ensureStop.on = false
		if !cmd.NoEnsureStop {
			if err := KillExecCmd(cmd.Cmd); err != nil {
				// ignoring error: just best effort to stop process
			}
		}
	}
}

//----------

func RunCmdOutputs(cmd *Cmd) (sout []byte, serr []byte, _ error) {
	if cmd.Cmd.Stdout != nil {
		return nil, nil, fmt.Errorf("stdout already set")
	}
	if cmd.Cmd.Stderr != nil {
		return nil, nil, fmt.Errorf("stderr already set")
	}
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Cmd.Stdout = &stdoutBuf
	cmd.Cmd.Stderr = &stderrBuf
	err := cmd.Run()
	return stdoutBuf.Bytes(), stderrBuf.Bytes(), err
}

// Adds stderr to err if it happens.
func RunCmdStdoutAndStderrInErr(cmd *Cmd) ([]byte, error) {
	bout, berr, err := RunCmdOutputs(cmd)
	if err != nil {
		serr := strings.TrimSpace(string(berr))
		if serr != "" {
			err = fmt.Errorf("%w: stderr(%v)", err, serr)
		}
		return nil, err
	}
	return bout, nil
}

func RunCmdStdoutAndStderrInErr2(ctx context.Context, dir string, args []string, env []string) ([]byte, error) {
	cmd := NewCmd(ctx, args...)
	cmd.Dir = dir
	cmd.Env = env
	return RunCmdStdoutAndStderrInErr(cmd)
}

func RunCmdStdoutAndStderrInErr3(ctx context.Context, dir string, args []string, env []string, stdin io.ReadCloser) ([]byte, error) {
	cmd := NewCmd(ctx, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin = stdin
	return RunCmdStdoutAndStderrInErr(cmd)
}

//----------
