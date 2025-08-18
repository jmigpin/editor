//go:build windows

package osutil

import (
	"fmt"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

//----------

const EscapeRune = '^'

//----------

func ProcAttrSetDefaults(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &windows.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_NO_WINDOW

	// requires a group kill at the end if started with a "shell" (ex: win cmd). cmd.process.kill will only kill the shell and not the proc
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_NEW_PROCESS_GROUP
}

func ProcAttrSetControllingTTY(cmd *exec.Cmd) {
	// NOOP
}

func ProcKill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return fmt.Errorf("process is nil")
	}

	// fails if started with CREATE_NEW_PROCESS_GROUP
	//return cmd.Process.Kill()

	// send CTRL_BREAK to the group (pid == group id)
	// TODO: fails to work
	//return windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(cmd.Process.Pid))

	// child/groups processes
	pid := fmt.Sprintf("%v", cmd.Process.Pid)
	c := exec.Command("taskkill", "/T", "/F", "/PID", pid)
	return c.Run()
}

//----------

func ShellCmdArgs(args ...string) []string {
	return append([]string{"cmd", "/C"}, args...)
}
func ShellScriptArgs(args ...string) []string {
	return ShellCmdArgs(args...)
}

//----------

func ExecName(name string) string {
	return name + ".exe"
}

//----------

func FsCaseFilename(filename string) (string, error) {
	namep, err := syscall.UTF16PtrFromString(filename)
	if err != nil {
		return "", err
	}

	// Short paths can be longer than long paths, and unicode
	buf := make([]uint16, 4*len(filename))
	bufLen := len(buf) * 2 // in bytes

	short := buf
	n, err := syscall.GetShortPathName(namep, &short[0], uint32(bufLen))
	if err != nil {
		return "", err
	}
	if int(n) > bufLen {
		return "", fmt.Errorf("short buffer too short: %v vs %v", n, bufLen)
	}

	long := make([]uint16, len(buf))
	n, err = syscall.GetLongPathName(&short[0], &long[0], uint32(bufLen))
	if err != nil {
		return "", err
	}
	if int(n) > bufLen {
		return "", fmt.Errorf("long buffer too short: %v vs %v", n, bufLen)
	}

	longStr := syscall.UTF16ToString(long)
	return longStr, nil
}
