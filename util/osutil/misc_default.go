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

func ProcAttrSetDefaults(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &unix.SysProcAttr{}
	}
	// set new group
	cmd.SysProcAttr.Setpgid = true
}

func ProcAttrSetControllingTTY(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &unix.SysProcAttr{}
	}
	// Setsid=true: the child does setsid(). It becomes session leader, gets a new process group, and drops any existing controlling TTY. This is required before it can receive a new controlling terminal.
	cmd.SysProcAttr.Setsid = true
	cmd.SysProcAttr.Setpgid = false // ensure or it can fail
	// Setctty=true: after setsid(), the kernel does TIOCSCTTY on fd 0 of the child. Because fd 0/1/2 is wired to the PTY slave, this makes that PTY the child’s controlling TTY.
	cmd.SysProcAttr.Setctty = true
}

func ProcKill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return fmt.Errorf("process is nil")
	}
	// negative pid (but !=-1) sends signals to the process group
	return unix.Kill(-cmd.Process.Pid, unix.SIGKILL)
}

//----------

// deals correctly with args that contain spaces
// scripting possible with args[0] set to "<script>; true"
func ShellCmdArgs(args ...string) []string {
	script := args[0] + " \"$@\""
	return append([]string{"sh", "-c", script, "sh"}, args[1:]...)
}

// allows scripting, but can have issues on args with spaces:
// ex: "echo a > b.txt"
// ex: "VAR_1=2 prog arg1 arg2"
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
