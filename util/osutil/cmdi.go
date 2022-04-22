package osutil

import (
	"context"
	"os/exec"

	"github.com/jmigpin/editor/util/iout"
)

type CmdI interface {
	Cmd() *exec.Cmd
	Start() error
	Wait() error
}

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

type SetSidCmd struct {
	CmdI
	done context.CancelFunc
}

func NewSetSidCmd(ctx context.Context, cmdi CmdI) *SetSidCmd {
	c := &SetSidCmd{CmdI: cmdi}
	SetupExecCmdSysProcAttr(c.CmdI.Cmd())

	ctx2, cancel := context.WithCancel(ctx)
	c.done = cancel // clear resources
	go func() {
		select {
		case <-ctx2.Done():
			// either the cmd is running and should be killed, or it reached the end and this goroutine should be unblocked
			_ = KillExecCmd(c.CmdI.Cmd()) // effective kill
		}
	}()
	return c
}
func (c *SetSidCmd) Start() error {
	if err := c.CmdI.Start(); err != nil {
		c.done()
		return err
	}
	return nil
}
func (c *SetSidCmd) Wait() error {
	defer c.done()
	return c.CmdI.Wait()
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
	if err := c.CmdI.Start(); err != nil {
		c.unpause()
		return err
	}
	c.callback(c)
	c.unpause()
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
