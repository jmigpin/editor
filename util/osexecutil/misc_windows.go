// +build windows

package osexecutil

import (
	"errors"
	"os/exec"
)

func SetupExecCmdSysProcAttr(cmd *exec.Cmd) {
	// todo
}

func KillExecCmd(cmd *exec.Cmd) error {
	return errors.New("todo: windows implementation")
}
