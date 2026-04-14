package core

import (
	"fmt"
	"os"
	"strings"

	"github.com/jmigpin/editor/ui"
)

type ExternalCmdMode int

const (
	ExternalCmdModeShellScript ExternalCmdMode = iota
	ExternalCmdModeShellArgs
)

func StartTerminalEmu(ed *Editor, dir string, rowPos *ui.RowPos, shellCmd string, shellArgs []string) error {
	info := ed.ReadERowInfo(dir)
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %v", dir)
	}

	erow := NewBasicERow(info, rowPos)

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	mode := ExternalCmdModeShellScript
	cargs := []string{shell}
	toolbarCmd := shell
	if len(shellArgs) > 0 {
		mode = ExternalCmdModeShellArgs
		cargs = shellArgs
		toolbarCmd = strings.Join(shellArgs, " ")
	} else if shellCmd != "" {
		cargs = []string{shellCmd}
		toolbarCmd = shellCmd
	}

	toolbarCmd = strings.ReplaceAll(toolbarCmd, "|", "\\|")
	erow.ToolbarSetStrAfterNameClearHistory(" | $font=auto | $terminal=emu | Stop | " + toolbarCmd)
	ExternalCmd2(erow, nil, cargs, nil, nil, mode)
	return nil
}
