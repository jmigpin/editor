package core

////godebug:annotatefile

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
)

func ExternalCmd(erow *ERow, part *toolbarparser.Part) {
	cargs := cmdPartArgs(part)
	ExternalCmdFromArgs(erow, cargs, nil)
}

func ExternalCmdFromArgs(erow *ERow, cargs []string, fend func(error)) {
	if erow.Info.IsFileButNotDir() {
		externalCmdFromFile(erow, cargs, fend)
	} else if erow.Info.IsDir() {
		env := populateEnvVars(erow, cargs)
		externalCmdFromDir(erow, cargs, fend, env)
	} else {
		erow.Ed.Errorf("unable to run external cmd for erow: %v", erow.Info.Name())
	}
}

//----------

// create a row with the file dir and run the cmd
func externalCmdFromFile(erow *ERow, cargs []string, fend func(error)) {
	dir := filepath.Dir(erow.Info.Name())

	info := erow.Ed.ReadERowInfo(dir)
	rowPos := erow.Row.PosBelow()
	erow2 := NewBasicERow(info, rowPos)

	env := populateEnvVars(erow, cargs)

	externalCmdFromDir(erow2, cargs, fend, env)
}

//----------

func populateEnvVars(erow *ERow, cargs []string) []string {
	// Can't use os.Expand() to replace (and show the values in cargs) since the idea is for the variable to be available in scripting if wanted.

	// supported environ vars
	m := map[string]func() string{
		"edName": erow.Info.Name, // filename
		"edDir":  erow.Info.Dir,  // directory
		"edFileOffset": func() string { // filename + offset "filename:#123"
			return cmdVar_getFileOffset(erow)
		},
		"edLine": func() string { // line only
			return cmdVar_getLine(erow)
		},

		// Deprecated: use $edFileOffset (just renamed)
		"edPosOffset": func() string { // filename + offset "filename:#123"
			return cmdVar_getFileOffset(erow)
		},
	}
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

	// add toolbar defined vars
	vmap := toolbarparser.ParseVars(&erow.TbData)
	for k, v := range vmap {
		if strings.HasPrefix(k, "$") {
			u := k[1:]
			env = append(env, u+"="+v)
		}
	}

	return env
}

func cmdVar_getFileOffset(erow *ERow) string {
	if !erow.Info.IsFileButNotDir() {
		return ""
	}
	offset := erow.Row.TextArea.CursorIndex()
	posOffset := fmt.Sprintf("%v:#%v", erow.Info.Name(), offset)
	return posOffset
}

func cmdVar_getLine(erow *ERow) string {
	if !erow.Info.IsFileButNotDir() {
		return ""
	}
	ta := erow.Row.TextArea
	l, _, err := parseutil.IndexLineColumn(ta.RW(), ta.CursorIndex())
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%v", l)
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
	cmd := osutil.NewCmd(ctx, cargs...)
	cmd.Dir = erow.Info.Name()
	cmd.Env = env

	if err := cmd.SetupStdio(rw, rw, rw); err != nil {
		return err
	}

	// output pid before any output
	cmd.PreOutputCallback = func() {
		cargsStr := strings.Join(cargs, " ")
		fmt.Fprintf(rw, "# pid %d: %s\n", cmd.Process.Pid, cargsStr)
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Wait()
}

//----------

func cmdPartArgs(part *toolbarparser.Part) []string {
	var u []string
	for _, a := range part.Args {
		// TODO: auto add escapes for spaces in case of "some arg"?
		//s := a.UnquotedStr()
		s := a.Str()
		s = parseutil.RemoveEscapesEscapable(s, osutil.EscapeRune, "|")
		u = append(u, s)
	}
	return osutil.ShellRunArgs(u...)
}

//----------
