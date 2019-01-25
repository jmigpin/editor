// +build windows

package osutil

import (
	"errors"
	"os"
	"os/exec"
)

//----------

var FilenameEscapeRunes = " %?<>()^"

//----------

func SetupExecCmdSysProcAttr(cmd *exec.Cmd) {
}

func KillExecCmd(cmd *exec.Cmd) error {
	return errors.New("todo: windows implementation")
}

//----------

func ShellRunArgs(args ...string) []string {
	return args
	//return []string{"bash", "-exec", strings.Join(args, " ")}
}

//----------

func GoExec() string {
	return "go.exe"
}

//----------

func HomeEnvVar() string {
	return os.Getenv("USERPROFILE")
}
