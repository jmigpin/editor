package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmigpin/editor/core/parseutil"
	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/jmigpin/editor/util/osutil"
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

	// funcs that populate env vars

	getPosOffset := func() string {
		if !erow.Info.IsFileButNotDir() {
			return ""
		}
		offset := erow.Row.TextArea.TextCursor.Index()
		posOffset := fmt.Sprintf("%v:#%v", erow.Info.Name(), offset)
		return posOffset
	}

	getLine := func() string {
		if !erow.Info.IsFileButNotDir() {
			return ""
		}
		tc := erow.Row.TextArea.TextCursor
		l, _, err := parseutil.IndexLineColumn(tc.RW(), tc.Index())
		if err != nil {
			return ""
		}
		return fmt.Sprintf("%v", l)
	}

	// supported env vars
	m := map[string]func() string{
		"edName":      erow.Info.Name, // filename
		"edDir":       erow.Info.Dir,  // directory
		"edPosOffset": getPosOffset,   // filename + offset
		"edLine":      getLine,        // line only
	}

	// populate env vars if detected
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

//----------

func externalCmdDir(erow *ERow, cargs []string, fend func(error), env []string) {
	if !erow.Info.IsDir() {
		panic("not a directory")
	}
	erow.Exec.Start(func(ctx context.Context, w io.Writer) error {
		// cleanup row content
		erow.Ed.UI.RunOnUIGoRoutine(func() {
			erow.Row.TextArea.SetStrClearHistory("")
			erow.Row.TextArea.ClearPos()
		})

		err := externalCmdDir2(erow, cargs, env, ctx, w)
		if fend != nil {
			fend(err)
		}
		return err
	})
}

func externalCmdDir2(erow *ERow, cargs []string, env []string, ctx context.Context, w io.Writer) error {
	// prepare cmd exec
	cmd := osutil.ExecCmdCtxWithAttr(ctx, cargs[0], cargs[1:]...)
	cmd.Dir = erow.Info.Name()
	cmd.Env = env
	cmd.Stdout = w
	cmd.Stderr = w

	// TODO: pty tests
	//f, err := pty.Start(cmd)
	//if err != nil {
	//	return err
	//}
	//cmd.Stdin = f

	// run command
	err := cmd.Start()
	if err != nil {
		return err
	}

	// ensure kill to child processes on function exit (failsafe)
	go func() {
		select {
		case <-ctx.Done():
			if err := osutil.KillExecCmd(cmd); err != nil {
				fmt.Fprintf(w, "# error: kill: %v\n", err)
			}
		}
	}()

	// TODO: ensure first output is pid with altered writer
	// output pid
	cargsStr := strings.Join(cargs, " ")
	fmt.Fprintf(w, "# pid %d: %s\n", cmd.Process.Pid, cargsStr)

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
