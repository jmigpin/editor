// +build windows

package osutil

import (
	"errors"
	"os/exec"
	"strings"
	"syscall"
)

//----------

const EscapeRune = '^'

//----------

func SetupExecCmdSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

func KillExecCmd(cmd *exec.Cmd) error {
	return errors.New("todo: windows implementation")
}

//----------

func ShellRunArgs(args ...string) []string {
	return []string{"powershell", "-c", strings.Join(args, " ")}
}

//----------

func ExecName(name string) string {
	return name + ".exe"
}
