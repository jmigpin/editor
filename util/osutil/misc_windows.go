// +build windows

package osutil

import (
	"errors"
	"os/exec"
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
	//return []string{"bash", "-exec", strings.Join(args, " ")}

	// doesn't work: if the program was compiled with "-H=windowsgui", each exec.Command will spawn a new console window.
	//cmdPath := "C:\\Windows\\system32\\cmd.exe"
	//return []string{cmdPath, "/c", strings.Join(args, " ")}

	return args
}

//----------

func GoExec() string {
	return ExecName("go")
}
func ExecName(name string) string {
	return name + ".exe"
}
