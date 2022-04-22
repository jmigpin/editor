//go:build windows
// +build windows

package osutil

import (
	"fmt"
	"os/exec"

	"golang.org/x/sys/windows"
)

//----------

const EscapeRune = '^'

//----------

func SetupExecCmdSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &windows.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

func KillExecCmd(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return fmt.Errorf("process is nil")
	}
	return cmd.Process.Kill()

	// TODO: child/groups processes?
	//pid := fmt.Sprintf("%v", cmd.Process.Pid)
	//c := exec.Command("taskkill", "/T", "/F", "/PID", pid)
	//return c.Run()
}

//----------

func ShellRunArgs(args ...string) []string {
	return append([]string{"cmd", "/C"}, args...)
}

//----------

func ExecName(name string) string {
	return name + ".exe"
}
