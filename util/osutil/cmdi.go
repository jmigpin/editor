package osutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/jmigpin/editor/util/iout"
)

//godebug:annotatefile

type CmdI interface {
	Cmd() *exec.Cmd
	Start() error
	Wait() error
}

//----------
//----------
//----------

func NewCmdI(cmd *exec.Cmd) CmdI {
	return NewBasicCmd(cmd)
}

func NewCmdI2(ctx context.Context, args ...string) CmdI {
	// NOTE: not using exec.CommandContext because the ctx is dealt with in NewCtxCmd
	cmd := exec.Command(args[0], args[1:]...)

	return NewCmdI3(ctx, cmd)
}

func NewCmdI3(ctx context.Context, cmd *exec.Cmd) CmdI {
	c := NewCmdI(cmd)
	c = NewNoHangPipeCmd(c, true, true, true)
	if ctx != nil {
		c = NewCtxCmd(ctx, c)
	}
	c = NewShellCmd(c)
	return c
}

//----------
//----------
//----------

type BasicCmd struct {
	cmd *exec.Cmd
}

func NewBasicCmd(cmd *exec.Cmd) *BasicCmd {
	return &BasicCmd{cmd: cmd}
}
func (c *BasicCmd) Cmd() *exec.Cmd {
	return c.cmd
}
func (c *BasicCmd) Start() error {
	return c.cmd.Start()
}
func (c *BasicCmd) Wait() error {
	return c.cmd.Wait()
}

//----------
//----------
//----------

type ShellCmd struct {
	CmdI
}

func NewShellCmd(cmdi CmdI) *ShellCmd {
	c := &ShellCmd{CmdI: cmdi}
	cmd := c.CmdI.Cmd()
	cmd.Args = ShellRunArgs(cmd.Args...)

	// update cmd.path with shell executable
	name := cmd.Args[0]
	cmd.Path = name
	if lp, err := exec.LookPath(name); err == nil {
		cmd.Path = lp
	}
	// TODO: review
	// set to nil (ex: exec.command can set this at init in case it doesn't find the exec)
	cmd.Err = nil

	return c
}

//----------
//----------
//----------

// Old note: explanations on possible hangs.
// https://github.com/golang/go/issues/18874#issuecomment-277280139

//CtxCmd behaviour is somewhat equivalent to:
// 	cmd := exec.CommandContext(ctx, args[0], args[1:]...))
// 	cmd.WaitDelay = X * time.Second
//but it has a custom error msg and sends a term signal that can include the process group. Beware that if using this, the exec.cmd should probably not be started with exec.commandcontext, since that will have the ctx cancel run first (before this handler) and when it gets here the process is already canceled.

type CtxCmd struct {
	CmdI
	ctx context.Context
}

func NewCtxCmd(ctx context.Context, cmdi CmdI) *CtxCmd {
	c := &CtxCmd{CmdI: cmdi, ctx: ctx}

	SetupExecCmdSysProcAttr(c.CmdI.Cmd())

	return c
}
func (c *CtxCmd) Start() error {
	return c.CmdI.Start()
}
func (c *CtxCmd) Wait() error {
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- c.CmdI.Wait()
	}()
	select {
	case err := <-waitCh:
		return err
	case <-c.ctx.Done():
		_ = KillExecCmd(c.CmdI.Cmd())

		// wait for the possibility of wait returning after kill
		timeout := 3 * time.Second
		select {
		case err := <-waitCh:
			return err
		case <-time.After(timeout):
			// warn about the process not returning
			s := fmt.Sprintf("termination timeout (%v): process has not returned from wait (ex: a subprocess might be keeping a file descriptor open). Beware that these processes might produce output visible here.\n", timeout)
			//c.printf(s)

			// exit now (leaks waitCh go routine)
			//return c.ctx.Err()
			return errors.New(s)

			//// wait forever
			//return <-waitCh
		}
	}
}
func (c *CtxCmd) printf(f string, args ...any) {
	cmd := c.CmdI.Cmd()
	if cmd.Stderr == nil {
		return
	}
	fmt.Fprintf(cmd.Stderr, "# ctxcmd: "+f, args...)
}

//----------
//----------
//----------

type NoHangPipeCmd struct {
	CmdI
	doIn, doOut, doErr bool
	stdin              io.WriteCloser
	//stdout             io.ReadCloser
	//stderr             io.ReadCloser
	//outPipes           sync.WaitGroup // stdout/stderr pipe wait
}

func NewNoHangPipeCmd(cmdi CmdI, doIn, doOut, doErr bool) *NoHangPipeCmd {
	return &NoHangPipeCmd{CmdI: cmdi, doIn: doIn, doOut: doOut, doErr: doErr}
}
func (c *NoHangPipeCmd) Start() error {
	cmd := c.Cmd()
	if c.doIn && cmd.Stdin != nil {
		r := cmd.Stdin
		cmd.Stdin = nil // cmd wants nil here
		wc, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		c.stdin = wc
		go func() {
			_, _ = io.Copy(wc, r)
			_ = wc.Close()
		}()
	}
	//if c.doOut && cmd.Stdout != nil {
	//	w := cmd.Stdout
	//	cmd.Stdout = nil // cmd wants nil here
	//	rc, err := cmd.StdoutPipe()
	//	if err != nil {
	//		return err
	//	}
	//	c.stdout = rc
	//	c.outPipes.Add(1)
	//	go func() {
	//		defer c.outPipes.Done()
	//		_, _ = io.Copy(w, rc)
	//		_ = rc.Close()
	//	}()
	//}
	//if c.doErr && cmd.Stderr != nil {
	//	w := cmd.Stderr
	//	cmd.Stderr = nil // cmd wants nil here
	//	rc, err := cmd.StderrPipe()
	//	if err != nil {
	//		return err
	//	}
	//	c.stderr = rc
	//	c.outPipes.Add(1)
	//	go func() {
	//		defer c.outPipes.Done()
	//		_, _ = io.Copy(w, rc)
	//		_ = rc.Close()
	//	}()
	//}
	return c.CmdI.Start()
}

//func (c *NoHangPipeCmd) Wait() error {
//	//c.outPipes.Wait() // wait for stdout/stderr pipes before calling wait
//	return c.CmdI.Wait()
//}

// some commands will not exit unless the stdin is closed, allow access
func (c *NoHangPipeCmd) CloseStdin() error {
	if c.stdin != nil {
		return c.stdin.Close()
	}
	return nil
}

//----------
//----------
//----------

// ex: usefull to print something before any cmd output is printed
type PausedWritersCmd struct {
	CmdI
	callback func(CmdI)
	stdout   *iout.PausedWriter
	stderr   *iout.PausedWriter
}

func NewPausedWritersCmd(cmdi CmdI, cb func(CmdI)) *PausedWritersCmd {
	c := &PausedWritersCmd{CmdI: cmdi, callback: cb}
	return c
}
func (c *PausedWritersCmd) Start() error {
	defer c.unpause()
	cmd := c.CmdI.Cmd()
	if cmd.Stdout != nil {
		c.stdout = iout.NewPausedWriter(cmd.Stdout)
	}
	if cmd.Stderr != nil {
		c.stderr = iout.NewPausedWriter(cmd.Stderr)
	}
	if err := c.CmdI.Start(); err != nil {
		return err
	}
	c.callback(c)
	return nil
}
func (c *PausedWritersCmd) Wait() error {
	c.unpause()
	return c.CmdI.Wait()
}
func (c *PausedWritersCmd) unpause() {
	if c.stdout != nil {
		c.stdout.Unpause()
	}
	if c.stderr != nil {
		c.stderr.Unpause()
	}
}

//----------
//----------
//----------

func RunCmdI(ci CmdI) error {
	if err := ci.Start(); err != nil {
		return err
	}
	return ci.Wait()
}
func RunCmdIOutputs(c CmdI) (sout []byte, serr []byte, _ error) {
	obuf := &bytes.Buffer{}
	ebuf := &bytes.Buffer{}

	cmd := c.Cmd()
	if cmd.Stdout != nil {
		return nil, nil, fmt.Errorf("stdout already set")
	}
	if cmd.Stderr != nil {
		return nil, nil, fmt.Errorf("stderr already set")
	}
	cmd.Stdout = obuf
	cmd.Stderr = ebuf

	err := RunCmdI(c)
	return obuf.Bytes(), ebuf.Bytes(), err
}
func RunCmdICombineStdoutStderr(c CmdI) ([]byte, error) {
	buf := &bytes.Buffer{}

	cmd := c.Cmd()
	if cmd.Stdout != nil {
		return nil, fmt.Errorf("stdout already set")
	}
	if cmd.Stderr != nil {
		return nil, fmt.Errorf("stderr already set")
	}
	cmd.Stdout = buf
	cmd.Stderr = buf

	err := RunCmdI(c)
	return buf.Bytes(), err
}
func RunCmdICombineStderrErr(c CmdI) ([]byte, error) {
	bout, berr, err := RunCmdIOutputs(c)
	if err != nil {
		serr := strings.TrimSpace(string(berr))
		if serr != "" {
			err = fmt.Errorf("%w: stderr(%v)", err, serr)
		}
		return nil, err
	}
	return bout, nil
}

//----------

func RunCmd(ctx context.Context, dir string, args ...string) ([]byte, error) {
	return RunCmdStdin(ctx, dir, nil, args...)
}
func RunCmdStdin(ctx context.Context, dir string, rd io.Reader, args ...string) ([]byte, error) {
	c := NewCmdI2(ctx, args...)
	c.Cmd().Dir = dir
	c.Cmd().Stdin = rd
	return RunCmdICombineStderrErr(c)
}
