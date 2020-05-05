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

//godebug:annotatefile

type Cmd struct {
	*exec.Cmd
	ctx         context.Context
	cancelCtx   context.CancelFunc
	setupCalled bool

	PreOutputCallback func()

	NoEnsureStop bool
	ensureStop   struct {
		sync.Mutex
		off bool
	}

	closers []io.Closer

	copy struct {
		fns     []func()
		closers []io.Closer
	}
}

// If Start() is not called, Cancel() must be called to clear resources.
func NewCmd(ctx context.Context, args ...string) *Cmd {
	ctx2, cancel := context.WithCancel(ctx)
	c := exec.CommandContext(ctx2, args[0], args[1:]...) // panic on empty args
	cmd := &Cmd{Cmd: c, ctx: ctx2, cancelCtx: cancel}
	return cmd
}

//----------

// can be called before a Start(), clears all possible resources
func (cmd *Cmd) Cancel() {
	cmd.cancelCtx()
	cmd.closeCopyClosers()
}

//----------

// If Start() returns no error, Wait() must be called to clear resources.
func (cmd *Cmd) Start() error {
	if err := cmd.start2(); err != nil {
		cmd.Cancel()
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

	// Ensure callback is called before the first stdout/stderr write (works since the process takes longer to launch and write back, but in theory this could be called after some output comes out). Always works if stdin/out/err was setup with SetupStdio() since the copy loop starts after the callback.
	if cmd.PreOutputCallback != nil {
		cmd.PreOutputCallback()
	}

	cmd.runCopyFns()

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
		cmd.disableEnsureStop() // no need to kill process anymore
		cmd.Cancel()            // clear resources
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

func (cmd *Cmd) disableEnsureStop() {
	cmd.ensureStop.Lock()
	defer cmd.ensureStop.Unlock()
	cmd.ensureStop.off = true
}

func (cmd *Cmd) ensureStopNow() {
	cmd.ensureStop.Lock()
	defer cmd.ensureStop.Unlock()
	if !cmd.ensureStop.off {
		cmd.ensureStop.off = true
		if !cmd.NoEnsureStop {
			if err := KillExecCmd(cmd.Cmd); err != nil {
				// ignoring error: just best effort to stop process
			}
		}
	}
}

//----------

func (cmd *Cmd) SetupStdio(ir io.Reader, ow, ew io.Writer) error {
	err := cmd.setupStdio2(ir, ow, ew)
	if err != nil {
		cmd.closeCopyClosers()
	}
	return err
}

func (cmd *Cmd) setupStdio2(ir io.Reader, ow, ew io.Writer) error {
	// setup only once (don't allow f(w, nil) and later f(nil, w)
	if cmd.setupCalled {
		return fmt.Errorf("setup already called")
	}
	cmd.setupCalled = true

	// setup stdin
	if ir != nil {
		ipwc, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		cmd.addCopyCloser(ipwc)
		cmd.copy.fns = append(cmd.copy.fns, func() {
			defer ipwc.Close()
			io.Copy(ipwc, ir)
		})
	}

	// setup stdout
	cmd.Cmd.Stdout = ow

	// setup stderr
	cmd.Cmd.Stderr = ew
	return nil
}

//----------

func (cmd *Cmd) runCopyFns() {
	for _, fn := range cmd.copy.fns {
		// go fn() // Commented: will call the same fn twice (loop var)
		go func(fn2 func()) {
			fn2()
		}(fn)
	}
}

//----------

func (cmd *Cmd) addCopyCloser(c io.Closer) {
	cmd.copy.closers = append(cmd.copy.closers, c)
}

func (cmd *Cmd) closeCopyClosers() {
	for _, c := range cmd.copy.closers {
		c.Close()
	}
}

//----------
//----------
//----------

func RunCmdCombinedOutput(cmd *Cmd, rd io.Reader) ([]byte, error) {
	obuf := &bytes.Buffer{}
	if err := cmd.SetupStdio(rd, obuf, obuf); err != nil {
		return nil, err
	}
	err := cmd.Run()
	return obuf.Bytes(), err
}

func RunCmdOutputs(cmd *Cmd, rd io.Reader) (sout []byte, serr []byte, _ error) {
	obuf := &bytes.Buffer{}
	ebuf := &bytes.Buffer{}
	if err := cmd.SetupStdio(rd, obuf, ebuf); err != nil {
		return nil, nil, err
	}
	err := cmd.Run()
	return obuf.Bytes(), ebuf.Bytes(), err
}

// Adds stderr to err if it happens.
func RunCmdStdoutAndStderrInErr(cmd *Cmd, rd io.Reader) ([]byte, error) {
	bout, berr, err := RunCmdOutputs(cmd, rd)
	if err != nil {
		serr := strings.TrimSpace(string(berr))
		if serr != "" {
			err = fmt.Errorf("%w: stderr(%v)", err, serr)
		}
		return nil, err
	}
	return bout, nil
}
