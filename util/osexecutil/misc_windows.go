// +build windows

package osexecutil

import (
	"errors"
	"os/exec"
	"strings"
)

func SetupExecCmdSysProcAttr(cmd *exec.Cmd) {
	// todo
}

func KillExecCmd(cmd *exec.Cmd) error {
	return errors.New("todo: windows implementation")
}

//----------

func ShellRunArgs(args ...string) []string {
	//return args
	return []string{"bash", "-exec", strings.Join(args, " ")}
}
