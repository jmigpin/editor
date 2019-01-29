// +build !windows

package osutil

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
)

//----------

const EscapeRune = '\\'
const EscapeRunes = string(EscapeRune)

const FilenameEscapeRunes = " :%?<>()\\"

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

//----------

func GoExec() string {
	return ExecName("go")
}
func ExecName(name string) string {
	return name
}

//----------

func HomeEnvVar() string {
	return os.Getenv("HOME")

}
