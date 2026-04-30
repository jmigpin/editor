package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmigpin/editor/core/termemu"
	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
)

// if part is nil, cargs should be set
// if cargs is nil, part should be set
func ExternalCmd(erow *ERow, part *toolbarparser.Part, cargs []string, fend func(error), env []string) {
	ExternalCmd2(erow, part, cargs, fend, env, ExternalCmdModeShellScript)
}

// if part is nil, cargs should be set
// if cargs is nil, part should be set
func ExternalCmd2(erow *ERow, part *toolbarparser.Part, cargs []string, fend func(error), env []string, mode ExternalCmdMode) {
	// before toolbar vars to allow override
	if erow.termOpts.emuOpts.Mode.On() {
		env = append(env, termemu.TermEnv...)
	}

	if part != nil {
		env = append(env, toolbarVarsEnv(part)...)
		if cargs == nil {
			cargs = cmdPartArgs(part)
		}
	}

	env = append(env, detectedEdEnvVars(erow, cargs)...)

	switch {
	case erow.Info.IsDir():
		externalCmdFromDir(erow, cargs, fend, env, mode)
	case erow.Info.IsFileButNotDir():
		// create a row with the file dir and run the cmd
		dir := filepath.Dir(erow.Info.Name())
		info := erow.Ed.ReadERowInfo(dir)
		rowPos := erow.Row.PosBelow()
		erow2 := NewBasicERow(info, rowPos)
		externalCmdFromDir(erow2, cargs, fend, env, mode)
	default:
		erow.Ed.Errorf("unable to run external cmd for erow: %v", erow.Info.Name())
	}
}

//----------

func externalCmdFromDir(erow *ERow, cargs []string, fend func(error), env []string, mode ExternalCmdMode) {
	if !erow.Info.IsDir() {
		panic("not a directory")
	}
	fend2 := func(err error) {
		if fend != nil {
			fend(err)
		}
	}
	_, _ = erow.Exec.RunAsync(func(ctx context.Context, rw io.ReadWriter) error {
		err := externalCmdFromDir2(ctx, erow, cargs, env, rw, mode)
		fend2(err)
		return err
	})
}

func externalCmdFromDir2(ctx context.Context, erow *ERow, cargs []string, env []string, rw io.ReadWriter, mode ExternalCmdMode) error {

	// TODO: unify this code with godebug cmd call

	c := osutil.NewCmdI2(cargs)
	switch mode {
	case ExternalCmdModeShellScript:
		c = osutil.NewShellCmd(c, true)
	case ExternalCmdModeShellArgs:
		c = osutil.NewShellCmd(c, false)
	default:
		panic(fmt.Sprintf("unexpected external cmd mode: %v", mode))
	}

	// first, to run start() last and wrap everything in a pty
	if erow.termOpts.pty {
		ptyCmd := osutil.NewPtyCmd(c)
		c = ptyCmd
		// set pty in the textarea
		if erow.optTemu != nil {
			erow.optTemu.setPty(ptyCmd)
		}
	} else {
		c = osutil.NewNoHangPipeCmd(c)
	}

	c = newPausedWriter(c, cargs, rw)

	// last, to run wait() first, such that a ctx cancel sends a proc kill
	c = osutil.NewCtxCmd(ctx, c)

	if erow.termOpts.pty && erow.optTemu != nil {
		// run callback on start
		c = osutil.NewFuncsCmd(c,
			func(inner osutil.CmdI) error { // on start
				if err := inner.Start(); err != nil {
					return err
				}
				return erow.optTemu.onPtyStart()
			},
			nil,
		)
	}

	cmd := c.Cmd()
	cmd.Dir = erow.Info.Name()
	cmd.Env = append(os.Environ(), env...)
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

func newPausedWriter(c osutil.CmdI, cargs []string, w io.Writer) osutil.CmdI {
	printPid := func(c osutil.CmdI) {
		//argsStr := strings.Join(c.Cmd().Args, " ")
		argsStr := strings.Join(cargs, " ")
		fmt.Fprintf(w, "# pid %d: %s\n", c.Cmd().Process.Pid, argsStr)
	}
	return osutil.NewPausedWritersCmd(c, printPid)
}

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

func detectedEdEnvVars(erow *ERow, cargs []string) []string {
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
	env := []string{}
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

//----------

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
