package osutil

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/jmigpin/editor/util/iout"
)

type CmdI interface {
	Cmd() *exec.Cmd
	Start() error
	Wait() error
}

//----------

func NewCmdI(cmd *exec.Cmd) CmdI {
	return NewBasicCmd(cmd)
}

func RunCmdI(ci CmdI) error {
	if err := ci.Start(); err != nil {
		return err
	}
	return ci.Wait()
}

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

	return c
}

//----------

//CtxCmd behaviour is somewhat equivalent to:
// 	cmd := exec.CommandContext(ctx, args[0], args[1:]...))
// 	cmd.WaitDelay = X * time.Second
//but it has a custom error msg and sends a term signal that can include the process group. Beware that if using this, the exec.cmd should probably not be started with exec.commandcontext, since that will have the ctx cancel run first (before this handler) and when it gets here the process is already canceled.

type CtxCmd struct {
	CmdI
	ctx    context.Context
	waitCh chan error
}

func NewCtxCmd(ctx context.Context, cmdi CmdI) *CtxCmd {
	c := &CtxCmd{CmdI: cmdi, ctx: ctx}
	c.waitCh = make(chan error, 1)

	SetupExecCmdSysProcAttr(c.CmdI.Cmd())

	return c
}
func (c *CtxCmd) Start() error {
	if err := c.CmdI.Start(); err != nil {
		return err
	}
	go func() {
		c.waitCh <- c.CmdI.Wait()
	}()
	return nil
}
func (c *CtxCmd) Wait() error {
	select {
	case err := <-c.waitCh:
		return err
	case <-c.ctx.Done():
		_ = KillExecCmd(c.CmdI.Cmd())

		// wait for the possibility of wait returning after kill
		timeout := 3 * time.Second
		select {
		case err := <-c.waitCh:
			return err
		case <-time.After(timeout):
			// warn about the process not returning
			s := fmt.Sprintf("termination timeout (%v): process has not returned from wait (ex: a subprocess might be keeping a file descriptor open).\n", timeout)
			c.printf(s)

			//// exit now (leaks waitCh go routine)
			////return c.ctx.Err()
			//return fmt.Errorf(s)

			// wait forever
			return <-c.waitCh
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

// ex: usefull to print something before any cmd output is printed
type CallbackOnStartCmd struct {
	CmdI
	callback func(CmdI)
	stdout   *iout.PausedWriter
	stderr   *iout.PausedWriter
}

func NewCallbackOnStartCmd(cmdi CmdI, cb func(CmdI)) *CallbackOnStartCmd {
	c := &CallbackOnStartCmd{CmdI: cmdi, callback: cb}
	cmd := c.CmdI.Cmd()
	if cmd.Stdout != nil {
		c.stdout = iout.NewPausedWriter(cmd.Stdout)
	}
	if cmd.Stderr != nil {
		c.stderr = iout.NewPausedWriter(cmd.Stderr)
	}
	return c
}
func (c *CallbackOnStartCmd) Start() error {
	defer c.unpause()
	if err := c.CmdI.Start(); err != nil {
		return err
	}
	c.callback(c)
	return nil
}
func (c *CallbackOnStartCmd) Wait() error {
	c.unpause()
	return c.CmdI.Wait()
}
func (c *CallbackOnStartCmd) unpause() {
	if c.stdout != nil {
		c.stdout.Unpause()
	}
	if c.stderr != nil {
		c.stderr.Unpause()
	}
}
