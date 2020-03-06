package core

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
		externalCmdFileButNotDir(erow, cargs, fend)
	} else if erow.Info.IsDir() {
		env := populateEnvVars(erow, cargs)
		externalCmdDir(erow, cargs, fend, env)
	} else {
		erow.Ed.Errorf("unable to run external cmd for erow: %v", erow.Info.Name())
	}
}

//----------

// create a row with the file dir and run the cmd
func externalCmdFileButNotDir(erow *ERow, cargs []string, fend func(error)) {
	dir := filepath.Dir(erow.Info.Name())

	info := erow.Ed.ReadERowInfo(dir)
	rowPos := erow.Row.PosBelow()
	erow2 := NewERow(erow.Ed, info, rowPos)

	env := populateEnvVars(erow, cargs)

	externalCmdDir(erow2, cargs, fend, env)
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

func externalCmdDir(erow *ERow, cargs []string, fend func(error), env []string) {
	if !erow.Info.IsDir() {
		panic("not a directory")
	}
	erow.Exec.RunAsync(func(ctx context.Context, w io.Writer) error {
		err := externalCmdDir2(ctx, erow, cargs, env, w)
		if fend != nil {
			fend(err)
		}
		return err
	})
}

func externalCmdDir2(ctx context.Context, erow *ERow, cargs []string, env []string, w io.Writer) error {
	cmd := osutil.NewCmd(ctx, cargs...)
	cmd.Dir = erow.Info.Name()
	cmd.Env = env
	if err := cmd.SetupStdio(nil, w, w); err != nil {
		return err
	}

	// output pid before any output
	cmd.PreOutputCallback = func() {
		cargsStr := strings.Join(cargs, " ")
		fmt.Fprintf(w, "# pid %d: %s\n", cmd.Process.Pid, cargsStr)
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Wait()
}

//----------

func cmdPartArgs(part *toolbarparser.Part) []string {
	//if partContainsEscapedPipes(part) {
	//	return shellCmdPartArgs(part)
	//}
	//return directCmdPartArgs(part)
	return shellCmdPartArgs(part)
}

//func partContainsEscapedPipes(part *toolbarparser.Part) bool {
//	for _, a := range part.Args {
//		if a.UnquotedStr() == "\\|" {
//			return true
//		}
//	}
//	return false
//}

//----------

func shellCmdPartArgs(part *toolbarparser.Part) []string {
	u := shellCmdPartArgsStr(part)
	return osutil.ShellRunArgs(u...)
}

func shellCmdPartArgsStr(part *toolbarparser.Part) []string {
	var u []string
	for _, a := range part.Args {
		s := a.Str()
		s = parseutil.RemoveEscapesEscapable(s, osutil.EscapeRune, "|")
		u = append(u, s)
	}
	return u
}

//----------

//func directCmdPartArgs(part *toolbarparser.Part) []string {
//	var u []string
//	for _, a := range part.Args {
//		s := a.UnquotedStr()
//		u = append(u, s)
//	}
//	return u
//}

//----------
