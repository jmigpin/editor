package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
)

func ExternalCmd(erow *ERow, part *toolbarparser.Part) {
	env := []string{}
	env = append(env, toolbarVarsEnv(part)...)
	cargs := cmdPartArgs(part)

	ExternalCmdFromArgs(erow, cargs, nil, env)
}

func ExternalCmdFromArgs(erow *ERow, cargs []string, fend func(error), env []string) {
	env = append(env, populateEdEnvVars(erow, cargs)...)

	switch {
	case erow.Info.IsDir():
		externalCmdFromDir(erow, cargs, fend, env)
	case erow.Info.IsFileButNotDir():
		// create a row with the file dir and run the cmd
		dir := filepath.Dir(erow.Info.Name())
		info := erow.Ed.ReadERowInfo(dir)
		rowPos := erow.Row.PosBelow()
		erow2 := NewBasicERow(info, rowPos)
		externalCmdFromDir(erow2, cargs, fend, env)
	default:
		erow.Ed.Errorf("unable to run external cmd for erow: %v", erow.Info.Name())
	}
}

//----------

func externalCmdFromDir(erow *ERow, cargs []string, fend func(error), env []string) {
	if !erow.Info.IsDir() {
		panic("not a directory")
	}
	erow.Exec.RunAsync(func(ctx context.Context, rw io.ReadWriter) error {
		err := externalCmdDir2(ctx, erow, cargs, env, rw)
		if fend != nil {
			fend(err)
		}
		return err
	})
}

func externalCmdDir2(ctx context.Context, erow *ERow, cargs []string, env []string, rw io.ReadWriter) error {

	printPid := func(c osutil.CmdI) {
		argsStr := strings.Join(c.Cmd().Args, " ")
		fmt.Fprintf(rw, "# pid %d: %s\n", c.Cmd().Process.Pid, argsStr)
	}

	c := osutil.NewCmdI2(ctx, cargs...)
	c = osutil.NewPausedWritersCmd(c, printPid)

	cmd := c.Cmd()
	cmd.Dir = erow.Info.Name()
	cmd.Env = env
	cmd.Stdin = rw
	cmd.Stdout = rw
	cmd.Stderr = rw

	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

//----------
//----------
//----------

func cmdPartArgs(part *toolbarparser.Part) []string {
	var u []string
	for _, a := range part.Args {
		s := a.String()
		if !parseutil.IsQuoted(s) {
			s = parseutil.RemoveEscapesEscapable(s, osutil.EscapeRune, "|")
		}
		u = append(u, s)
	}
	return u
}

//----------
//----------
//----------

func populateEdEnvVars(erow *ERow, cargs []string) []string {
	// Can't use os.Expand() to replace (and show the values in cargs) since the idea is for the variable to be available in scripting if wanted.

	// supported env vars
	m := map[string]func() string{
		"edName": erow.Info.Name, // filename
		"edDir":  erow.Info.Dir,  // directory
		"edFileOffset": func() string { // filename + offset "filename:#123"
			return cmdVar_edFileOffset(erow)
		},
		"edFileLine": func() string { // cursor line
			return cmdVar_edFileLine(erow)
		},
		"edFileWord": func() string {
			return cmdVar_edFileWord(erow)
		},
	}

	// Deprecated: allow continued usage
	m["edPosOffset"] = m["edFileOffset"]
	m["edLine"] = m["edFileLine"]

	// populate env vars only if detected
	env := os.Environ()
	for k, v := range m {
		for _, s := range cargs {
			if parseutil.DetectEnvVar(s, k) {
				env = append(env, k+"="+v())
				break
			}
		}
	}

	return env
}

func cmdVar_edFileOffset(erow *ERow) string {
	offset := erow.Row.TextArea.CursorIndex()
	posOffset := fmt.Sprintf("%v:#%v", erow.Info.Name(), offset)
	return posOffset
}

func cmdVar_edFileLine(erow *ERow) string {
	ta := erow.Row.TextArea
	l, _, err := parseutil.IndexLineColumn(ta.RW(), ta.CursorIndex())
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%v", l)
}

func cmdVar_edFileWord(erow *ERow) string {
	ta := erow.Row.TextArea
	b, _, err := iorw.WordAtIndex(ta.RW(), ta.CursorIndex())
	if err != nil {
		return ""
	}
	return string(b)
}

//----------

func toolbarVarsEnv(part *toolbarparser.Part) []string {
	data2 := *part.Data // copy

	// use data only up to the selected part
	for k, part2 := range data2.Parts {
		if part2 == part {
			data2.Parts = data2.Parts[:k+1]
			break
		}
	}

	env := []string{}
	vmap := toolbarparser.ParseVars(&data2)
	for k, v := range vmap {
		if strings.HasPrefix(k, "$") {
			u := k[1:]
			env = append(env, u+"="+v)
		}
	}
	return env
}
