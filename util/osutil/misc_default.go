//go:build !windows
// +build !windows

package osutil

import (
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/sys/unix"
)

//----------

const EscapeRune = '\\'

//----------

func SetupExecCmdSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &unix.SysProcAttr{Setsid: true}
}

func KillExecCmd(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return fmt.Errorf("process is nil")
	}
	// negative pid (but !=-1) sends signals to the process group
	return unix.Kill(-cmd.Process.Pid, unix.SIGKILL)
}

//----------

func ShellRunArgs(args ...string) []string {
	return []string{"sh", "-c", strings.Join(args, " ")}
}

//----------

func ExecName(name string) string {
	return name
}

//----------

func FsCaseFilename(filename string) (string, error) {
	return filename, nil
}
