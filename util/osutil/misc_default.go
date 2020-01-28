// +build !windows

package osutil

import (
	"os/exec"
	"strings"
	"syscall"
)

//----------

const EscapeRune = '\\'

//----------

func SetupExecCmdSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

func KillExecCmd(cmd *exec.Cmd) error {
	// negative pid (but !=-1) sends signals to the process group
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

//----------

func ShellRunArgs(args ...string) []string {
	return []string{"sh", "-c", strings.Join(args, " ")}
}

//----------

func ExecName(name string) string {
	return name
}
