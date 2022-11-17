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
