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
		toolbarCmd = shellArgsToolbarCmd(shellArgs)
	} else if shellCmd != "" {
		cargs = []string{shellCmd}
		toolbarCmd = shellCmd
	}

	if mode == ExternalCmdModeShellScript {
		if strings.ContainsAny(toolbarCmd, "\r\n") {
			toolbarCmd = "Run -script " + strconv.Quote(toolbarCmd)
		} else {
			toolbarCmd = strings.ReplaceAll(toolbarCmd, "|", "\\|")
		}
	}
	erow.ToolbarSetStrAfterNameClearHistory(" | $terminal=emu | $font=auto | Stop | " + toolbarCmd)
	ExternalCmd(erow, cargs, nil, nil, mode)
	return nil
}

//----------

func shellArgsToolbarCmd(args []string) string {
	plain := true
	for _, arg := range args {
		if arg == "" {
			plain = false
			break
		}
		for _, r := range arg {
			if !plainShellArgRune(r) {
				plain = false
				break
			}
		}
		if !plain {
			break
		}
	}
	if plain {
		return strings.Join(args, " ")
	}

	quotedArgs := []string{"Run", "-shellArgs", "--"}
	for _, arg := range args {
		quotedArgs = append(quotedArgs, strconv.Quote(arg))
	}
	return strings.Join(quotedArgs, " ")
}

func plainShellArgRune(r rune) bool {
	if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
		return true
	}
	switch r {
	case '_', '-', '.', '/', ':', '@', '%', '+', '=', ',':
		return true
	}
	return false
}
