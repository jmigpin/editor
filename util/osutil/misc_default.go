// +build !windows

package osutil

import (
	"os/exec"
	"strings"
	"syscall"
)

//----------

func SetupExecCmdSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

func KillExecCmd(cmd *exec.Cmd) error {
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

//----------

func ShellRunArgs(args ...string) []string {
	//return args
	return []string{"sh", "-c", strings.Join(args, " ")}
}
