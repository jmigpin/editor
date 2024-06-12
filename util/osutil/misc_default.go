//go:build !windows

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

func ShellCmdArgs(args ...string) []string {
	// NOTE: scripting possible with args[0] set to "<script>; true"
	script := args[0] + " \"$@\""
	return append([]string{"sh", "-c", script, "sh"}, args[1:]...)
}
func ShellScriptArgs(args ...string) []string {
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
