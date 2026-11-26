package core

import (
	"fmt"
	"os"
	"strconv"
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

	if mode == ExternalCmdModeShellScript && strings.ContainsAny(toolbarCmd, "\r\n") {
		toolbarCmd = "Run -script " + strconv.Quote(toolbarCmd)
	} else {
		toolbarCmd = strings.ReplaceAll(toolbarCmd, "|", "\\|")
	}
	erow.ToolbarSetStrAfterNameClearHistory(" | $terminal=emu | $font=auto | Stop | " + toolbarCmd)
	ExternalCmd(erow, cargs, nil, nil, mode)
	return nil
}
